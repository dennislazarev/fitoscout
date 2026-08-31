// Package app собирает приложение: хранилище, миграции, TLS, middleware
// и фоновые задачи (ADR-002 — один бинарник fitoscoutd).
package app

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"fitoscout/backend/internal/api"
	"fitoscout/backend/internal/auth"
	"fitoscout/backend/internal/config"
	"fitoscout/backend/internal/logging"
	"fitoscout/backend/internal/middleware"
	"fitoscout/backend/internal/services"
	"fitoscout/backend/internal/storage"
	"fitoscout/backend/internal/storage/mariadb"
	apptls "fitoscout/backend/internal/tls"
)

// App — экземпляр приложения fitoscoutd.
type App struct {
	version   string
	commit    string
	buildDate string
	cfg       *config.Config

	logger *logging.Logger
	db     *storage.DB
	server *http.Server
	cancel context.CancelFunc
}

// New создаёт экземпляр приложения. Тяжёлая инициализация — в Start().
func New(version, commit, buildDate string, cfg *config.Config) *App {
	return &App{
		version:   version,
		commit:    commit,
		buildDate: buildDate,
		cfg:       cfg,
	}
}

// Start инициализирует подсистемы и запускает сервер и фоновые задачи.
func (a *App) Start() error {
	a.logger = logging.New(logging.Config{
		Level:     a.cfg.Logging.Level,
		File:      a.cfg.Logging.Output,
		MaxSizeMB: a.cfg.Logging.MaxSizeMB,
		MaxFiles:  a.cfg.Logging.MaxFiles,
	})

	a.logger.Info("запуск приложения",
		logging.F("version", a.version),
		logging.F("commit", a.commit),
		logging.F("build_date", a.buildDate),
	)

	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel

	// Хранилище: MariaDB (задача #1, замена SQLite).
	db, err := storage.Open(ctx, storage.FromAppConfig(a.cfg), a.logger)
	if err != nil {
		return err
	}
	a.db = db

	// Миграции, встроенные в бинарник.
	migrator := storage.NewMigrator(db.GetDB(), a.logger)
	if err := migrator.Migrate(ctx); err != nil {
		return fmt.Errorf("ошибка применения миграций: %w", err)
	}

	// mTLS: клиентские сертификаты + CRL (ADR-006).
	tlsCfg, err := apptls.BuildConfig(a.cfg.TLS, a.logger)
	if err != nil {
		return err
	}

	rateLimiter := middleware.NewRateLimiter(a.cfg.Auth.RateLimitPerMin)

	// Создаём основной роутер API

	// Инициализация SyncService
	syncRepo := mariadb.NewSyncRepo(db.GetDB())
	moduleRepos := mariadb.NewModuleRepositoryMap(db.GetDB(), syncRepo)
	syncService := services.NewSyncService(syncRepo, moduleRepos)
	mainRouter := api.NewServer(a.version, a.commit, a.buildDate, db, syncService).Router()

	// healthz без mTLS (для мониторинга systemd/kubernetes)
	healthRouter := http.NewServeMux()
	healthRouter.HandleFunc("GET /api/v1/healthz", api.HandleHealthz)

	// Всё остальное под mTLS
	securedRouter := middleware.Chain(
		mainRouter,
		auth.RequireClientCertConfig(a.cfg.Roles.WebCN, a.cfg.Roles.AndroidCN),
		auth.VerifyClientHeader(a.cfg.Auth.ClientHeader),
	)

	// Объединяем роутеры
	finalRouter := http.NewServeMux()
	finalRouter.Handle("/api/v1/healthz", healthRouter)
	finalRouter.Handle("/", securedRouter)

	handler := middleware.Chain(
		finalRouter,
		middleware.RequestID,
		middleware.Logging(a.logger),
		middleware.Recovery(a.logger),
		rateLimiter.Middleware(),
	)

	addr := net.JoinHostPort(a.cfg.Server.Host, fmt.Sprintf("%d", a.cfg.Server.Port))
	a.server = &http.Server{
		Addr:         addr,
		Handler:      handler,
		TLSConfig:    tlsCfg,
		ReadTimeout:  time.Duration(a.cfg.Server.ReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(a.cfg.Server.WriteTimeoutSec) * time.Second,
	}

	go func() {
		a.logger.Info("HTTPS-сервер запущен", logging.F("addr", addr))
		if err := a.server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			a.logger.Error("ошибка HTTPS-сервера", logging.F("error", err.Error()))
		}
	}()

	// Автоочистка: фоновая горутина с ticker (ADR-002, фича 4, ADR-008).
	if a.cfg.Cleanup.Enabled {
		go func() {
			if err := a.RunAutoCleanup(ctx); err != nil {
				a.logger.Error("автоочистка завершена с ошибкой", logging.F("error", err.Error()))
			}
		}()
	} else {
		a.logger.Info("автоочистка отключена конфигурацией", logging.F("section", "cleanup"))
	}

	return nil
}

// Shutdown корректно останавливает сервер, фоновые задачи и пул подключений.
func (a *App) Shutdown() error {
	if a.logger != nil {
		a.logger.Info("остановка приложения")
	}
	if a.cancel != nil {
		a.cancel()
	}

	timeout := time.Duration(a.cfg.Server.ShutdownTimeoutSec) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if a.server != nil {
		if err := a.server.Shutdown(ctx); err != nil {
			a.logger.Error("ошибка остановки HTTPS-сервера", logging.F("error", err.Error()))
		}
	}

	if a.db != nil {
		if err := a.db.Close(); err != nil {
			return err
		}
	}
	return nil
}

package main

import (
        "fmt"
        "os"
        "os/signal"
        "syscall"

        "fitoscout/backend/internal/app"
        "fitoscout/backend/internal/config"
)

var (
        version   = "dev"
        commit    = "unknown"
        buildDate = "unknown"
)

func main() {
        fmt.Printf("fitoscout backend %s (commit: %s, built: %s)\n", version, commit, buildDate)

        configPath := "config.toml"
        if len(os.Args) > 1 {
                configPath = os.Args[1]
        }

        cfg, err := config.Load(configPath)
        if err != nil {
                fmt.Fprintf(os.Stderr, "ошибка загрузки конфига: %v\n", err)
                os.Exit(1)
        }

        fmt.Printf("конфиг загружен: %s\n", configPath)

        application := app.New(version, commit, buildDate, cfg)

        if err := application.Start(); err != nil {
                fmt.Fprintf(os.Stderr, "ошибка запуска приложения: %v\n", err)
                os.Exit(1)
        }

        quit := make(chan os.Signal, 1)
        signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
        <-quit

        if err := application.Shutdown(); err != nil {
                fmt.Fprintf(os.Stderr, "ошибка остановки приложения: %v\n", err)
                os.Exit(1)
        }

        fmt.Println("приложение остановлено корректно")
}
// Package tls собирает TLS-конфигурацию сервера с mTLS (ADR-006).
package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"fitoscout/backend/internal/auth"
	"fitoscout/backend/internal/config"
	"fitoscout/backend/internal/logging"
)

// BuildConfig собирает tls.Config: серверный сертификат, клиентский CA,
// обязательная проверка клиентских сертификатов и CRL (revoked.txt).
func BuildConfig(cfg config.TLSConfig, logger *logging.Logger) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("не удалось загрузить серверный сертификат: %w", err)
	}

	caPEM, err := os.ReadFile(cfg.CAFile)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать корневой сертификат: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("не удалось разобрать корневой сертификат %s", cfg.CAFile)
	}

	// CRL: отсутствие файла не фатально — считаем список пустым.
	revoked, err := LoadRevoked(cfg.RevokedFile)
	if err != nil {
		logger.Warn("список отозванных сертификатов не загружен",
			logging.F("error", err.Error()),
			logging.F("path", cfg.RevokedFile),
		)
		revoked = make(map[string]bool)
	} else {
		logger.Info("список отозванных сертификатов загружен",
			logging.F("revoked_count", len(revoked)),
		)
	}

	return &tls.Config{
		Certificates:          []tls.Certificate{cert},
		ClientCAs:             pool,
		ClientAuth:            tls.RequireAndVerifyClientCert,
		MinVersion:            tls.VersionTLS12,
		VerifyPeerCertificate: makeRevocationChecker(revoked, logger),
	}, nil
}

// makeRevocationChecker отклоняет рукопожатия с отозванными сертификатами:
// fingerprint сертификата ищется в CRL.
func makeRevocationChecker(revoked map[string]bool, logger *logging.Logger) func([][]byte, [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		if len(rawCerts) == 0 {
			return fmt.Errorf("клиентский сертификат не предоставлен")
		}
		cert, err := x509.ParseCertificate(rawCerts[0])
		if err != nil {
			return fmt.Errorf("не удалось разобрать клиентский сертификат: %w", err)
		}

		fingerprint := auth.FingerprintSHA256(cert)
		if revoked[fingerprint] {
			logger.Warn("запрос с отозванным сертификатом",
				logging.F("cn", cert.Subject.CommonName),
				logging.F("fingerprint", fingerprint),
			)
			return fmt.Errorf("клиентский сертификат отозван")
		}
		return nil
	}
}

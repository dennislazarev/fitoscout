package auth

import (
        "crypto/sha256"
        "crypto/x509"
        "encoding/hex"
        "net/http"
)

// PeerCertificate возвращает первый клиентский сертификат из TLS-соединения
// или nil, если сертификат не предоставлен.
func PeerCertificate(r *http.Request) *x509.Certificate {
        if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
                return nil
        }
        return r.TLS.PeerCertificates[0]
}

// ExtractCN возвращает Common Name клиентского сертификата
// (пустая строка, если сертификата нет).
func ExtractCN(r *http.Request) string {
        cert := PeerCertificate(r)
        if cert == nil {
                return ""
        }
        return cert.Subject.CommonName
}

// FingerprintSHA256 возвращает sha256-отпечаток сертификата (hex) —
// формат записей CRL в revoked.txt.
func FingerprintSHA256(cert *x509.Certificate) string {
        sum := sha256.Sum256(cert.Raw)
        return hex.EncodeToString(sum[:])
}
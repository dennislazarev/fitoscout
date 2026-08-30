package tls

import (
        "bufio"
        "fmt"
        "os"
        "strings"
)

// LoadRevoked читает CRL — текстовый файл с sha256-отпечатками
// отозванных сертификатов, по одному в строке.
// Пустые строки и комментарии (#) игнорируются.
func LoadRevoked(path string) (map[string]bool, error) {
        f, err := os.Open(path)
        if err != nil {
                return nil, fmt.Errorf("не удалось открыть список отозванных сертификатов %s: %w", path, err)
        }
        defer f.Close()

        revoked := make(map[string]bool)
        scanner := bufio.NewScanner(f)
        for scanner.Scan() {
                line := strings.TrimSpace(scanner.Text())
                if line == "" || strings.HasPrefix(line, "#") {
                        continue
                }
                revoked[strings.ToLower(line)] = true
        }
        if err := scanner.Err(); err != nil {
                return nil, fmt.Errorf("ошибка чтения списка отозванных сертификатов: %w", err)
        }
        return revoked, nil
}
package mariadb

import (
"testing"
)

func TestSanitizeDSN(t *testing.T) {
tests := []struct {
name     string
dsn      string
expected string
}{
{
name:     "простой пароль tcp",
dsn:      "fitoscout:secret@tcp(127.0.0.1:3306)/fitoscout",
expected: "fitoscout:***@tcp(127.0.0.1:3306)/fitoscout",
},
{
name:     "простой пароль unix",
dsn:      "fitoscout:secret@unix(/run/mysqld/mysqld.sock)/fitoscout",
expected: "fitoscout:***@unix(/run/mysqld/mysqld.sock)/fitoscout",
},
{
name:     "сложный пароль с @",
dsn:      "fitoscout:KfG6HrQ#~*yY?wG{Naty13mD@}@unix(/run/mysqld/mysqld10.sock)/fitoscout",
expected: "fitoscout:***@unix(/run/mysqld/mysqld10.sock)/fitoscout",
},
{
name:     "пароль с несколькими @",
dsn:      "user:p@ss@word@tcp(host:3306)/db",
expected: "user:***@tcp(host:3306)/db",
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
result := SanitizeDSN(tt.dsn)
if result != tt.expected {
t.Errorf("SanitizeDSN(%q) = %q, want %q", tt.dsn, result, tt.expected)
}
})
}
}
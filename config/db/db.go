package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
)

type Config struct {
	DBServer string
	DBPort   int
	DBName   string
	DBUser   string
	PwdOld   string
	PwdNew   string
}

func Load() *Config {
	return &Config{
		DBServer: get("DB_SERVER", "127.0.0.1"),
		DBPort:   getInt("DB_PORT", 1433),
		DBName:   get("DB_NAME", "NEW_POS"),
		DBUser:   get("DB_USER", "sa"),
		PwdOld:   os.Getenv("PASSWORD_OLD"),
		PwdNew:   os.Getenv("PASSWORD"),
	}
}

func (c *Config) Passwords() []string {
	out := make([]string, 0, 2)
	if c.PwdOld != "" {
		out = append(out, c.PwdOld)
	}
	if c.PwdNew != "" {
		out = append(out, c.PwdNew)
	}
	if len(out) == 0 {
		out = append(out, "")
	}
	return out
}

func buildConnString(server string, port int, dbName, user, password string) string {
	u := &url.URL{
		Scheme: "sqlserver",
		User:   url.UserPassword(user, password),
		Host:   fmt.Sprintf("%s:%d", server, port),
	}
	q := u.Query()
	q.Set("database", dbName)
	q.Set("encrypt", "disable") // sama seperti TS kamu: encrypt=false
	q.Set("TrustServerCertificate", "true")
	u.RawQuery = q.Encode()
	return u.String()
}

// ConnectAny mencoba PASSWORD_OLD lalu PASSWORD; kembalikan *sql.DB dan label kandidat yang berhasil.
func ConnectAny(ctx context.Context, cfg *Config) (*sql.DB, string, error) {
	passwords := cfg.Passwords()
	var lastErr error
	for i, pw := range passwords {
		dsn := buildConnString(cfg.DBServer, cfg.DBPort, cfg.DBName, cfg.DBUser, pw)
		db, err := sql.Open("sqlserver", dsn)
		if err != nil {
			lastErr = err
			continue
		}
		if err := ping(ctx, db); err != nil {
			_ = db.Close()
			lastErr = err
			continue
		}
		return db, fmt.Sprintf("index=%d", i), nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no password candidates")
	}
	return nil, "", fmt.Errorf("connect failed with all passwords: %w", lastErr)
}

func ping(ctx context.Context, db *sql.DB) error {
	c, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return db.PingContext(c)
}

func get(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}
func getInt(k string, def int) int {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

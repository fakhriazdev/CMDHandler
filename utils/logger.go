package utils

import (
	"log/slog"
	"os"
	_ "time"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
)

type Logger struct{ *slog.Logger }

// NewLogger: output berwarna, rapi, jam pendek, auto non-color kalau bukan TTY.
func NewLogger() *Logger {
	h := tint.NewHandler(os.Stdout, &tint.Options{
		Level:      slog.LevelInfo, // ubah ke LevelDebug kalau mau lebih rame
		TimeFormat: "15:04:05",     // 24h; ganti ke "2006-01-02 15:04:05" kalau mau lengkap
		NoColor:    !isatty.IsTerminal(os.Stdout.Fd()),
	})
	return &Logger{slog.New(h)}
}

// Helper manis (opsional)
func (l *Logger) OK(msg string, args ...any)    { l.Info("✅ "+msg, args...) }
func (l *Logger) Warn(msg string, args ...any)  { l.Logger.Warn("⚠️ "+msg, args...) }
func (l *Logger) Fail(msg string, args ...any)  { l.Logger.Error("💥 "+msg, args...) }
func (l *Logger) Fatal(msg string, args ...any) { l.Fail(msg, args...); os.Exit(1) }

// contoh: l.OK("sqlserver connected", "mode", which)

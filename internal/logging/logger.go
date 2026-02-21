package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/config"
)

type Logger struct {
	mu    sync.Mutex
	level int
	l     *log.Logger
	file  *os.File
}

const (
	levelDebug = iota
	levelInfo
	levelWarn
	levelError
)

func New(cfg *config.Config) (*Logger, error) {
	level, err := parseLevel(cfg.General.LogLevel)
	if err != nil {
		return nil, err
	}
	var writers []io.Writer
	writers = append(writers, os.Stdout)

	var file *os.File
	if cfg.General.LogFile != "" {
		if err := os.MkdirAll(filepath.Dir(cfg.General.LogFile), 0o755); err != nil {
			return nil, fmt.Errorf("create log dir: %w", err)
		}
		f, err := os.OpenFile(cfg.General.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, fmt.Errorf("open log file: %w", err)
		}
		file = f
		writers = append(writers, file)
	}

	return &Logger{
		level: level,
		l:     log.New(io.MultiWriter(writers...), "", log.Ldate|log.Ltime),
		file:  file,
	}, nil
}

func parseLevel(value string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return levelDebug, nil
	case "info", "":
		return levelInfo, nil
	case "warn", "warning":
		return levelWarn, nil
	case "error":
		return levelError, nil
	default:
		return 0, fmt.Errorf("invalid log level: %s", value)
	}
}

func (l *Logger) logf(level int, tag string, format string, args ...any) {
	if level < l.level {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.l.Printf("[%s] %s", tag, fmt.Sprintf(format, args...))
}

func (l *Logger) Debug(format string, args ...any) { l.logf(levelDebug, "DEBUG", format, args...) }
func (l *Logger) Info(format string, args ...any)  { l.logf(levelInfo, "INFO", format, args...) }
func (l *Logger) Warn(format string, args ...any)  { l.logf(levelWarn, "WARN", format, args...) }
func (l *Logger) Error(format string, args ...any) { l.logf(levelError, "ERROR", format, args...) }

func (l *Logger) Close() error {
	if l.file == nil {
		return nil
	}
	return l.file.Close()
}

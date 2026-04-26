package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/config"
)

type Logger struct {
	mu    *sync.Mutex
	level int
	l     *log.Logger
	file  *os.File
	name  string
}

const (
	levelDebug = iota
	levelInfo
	levelWarn
	levelError
)

var levelByName = map[string]int{
	"debug":   levelDebug,
	"info":    levelInfo,
	"warn":    levelWarn,
	"warning": levelWarn,
	"error":   levelError,
}

var levelNameByValue = map[int]string{
	levelDebug: "debug",
	levelInfo:  "info",
	levelWarn:  "warn",
	levelError: "error",
}

func New(cfg *config.Config) (*Logger, error) {
	level, err := ParseLevel(cfg.General.LogLevel)
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
		mu:    &sync.Mutex{},
		level: level,
		l:     log.New(io.MultiWriter(writers...), "", log.Ldate|log.Ltime|log.Lmicroseconds),
		file:  file,
		name:  "root",
	}, nil
}

func ParseLevel(value string) (int, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		normalized = "info"
	}
	level, ok := levelByName[normalized]
	if !ok {
		return 0, fmt.Errorf("invalid log level: %q (valid: %s)", value, strings.Join(ValidLevels(), "|"))
	}
	return level, nil
}

func NormalizeLevel(value string) (string, error) {
	level, err := ParseLevel(value)
	if err != nil {
		return "", err
	}
	return levelNameByValue[level], nil
}

func ValidLevels() []string {
	return []string{"debug", "info", "warn", "error"}
}

func (l *Logger) LevelName() string {
	if l == nil {
		return "unknown"
	}
	if name, ok := levelNameByValue[l.level]; ok {
		return name
	}
	return "unknown"
}

func (l *Logger) WithComponent(name string) *Logger {
	if l == nil {
		return nil
	}
	normalized := strings.TrimSpace(name)
	if normalized == "" {
		normalized = l.name
	}
	return &Logger{
		mu:    l.mu,
		level: l.level,
		l:     l.l,
		file:  l.file,
		name:  normalized,
	}
}

func (l *Logger) logf(level int, tag string, format string, args ...any) {
	if l == nil {
		return
	}
	if level < l.level {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	msg := fmt.Sprintf(format, args...)
	if strings.TrimSpace(l.name) != "" {
		l.l.Printf("[%s] [%s] %s", tag, l.name, msg)
		return
	}
	l.l.Printf("[%s] %s", tag, msg)
}

func (l *Logger) Debug(format string, args ...any) { l.logf(levelDebug, "DEBUG", format, args...) }
func (l *Logger) Info(format string, args ...any)  { l.logf(levelInfo, "INFO", format, args...) }
func (l *Logger) Warn(format string, args ...any)  { l.logf(levelWarn, "WARN", format, args...) }
func (l *Logger) Error(format string, args ...any) { l.logf(levelError, "ERROR", format, args...) }

// StartupBanner emits a clearly visible session-start block with a separator
// and explicit timestamp. It is always logged regardless of configured level.
func (l *Logger) StartupBanner(message string) {
	if l == nil {
		return
	}

	msg := strings.TrimSpace(message)
	if msg == "" {
		msg = "application start"
	}

	sep := strings.Repeat("=", 72)
	ts := time.Now().Format("2006-01-02 15:04:05 MST")

	l.mu.Lock()
	defer l.mu.Unlock()

	if strings.TrimSpace(l.name) != "" {
		l.l.Printf("[START] [%s] %s", l.name, sep)
		l.l.Printf("[START] [%s] %s | %s", l.name, msg, ts)
		l.l.Printf("[START] [%s] %s", l.name, sep)
		return
	}

	l.l.Printf("[START] %s", sep)
	l.l.Printf("[START] %s | %s", msg, ts)
	l.l.Printf("[START] %s", sep)
}

func (l *Logger) Close() error {
	if l.file == nil {
		return nil
	}
	return l.file.Close()
}

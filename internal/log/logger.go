package log

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// Logger defines minimal logging interface used across the project.
type Logger interface {
	Info(msg string, kv ...interface{})
	Warn(msg string, kv ...interface{})
	Error(msg string, kv ...interface{})
	Debug(msg string, kv ...interface{})
}

// Level represents log verbosity.
type Level int

const (
	ErrorLevel Level = iota
	WarnLevel
	InfoLevel
	DebugLevel
)

// String returns canonical lower-case representation.
func (l Level) String() string {
	switch l {
	case ErrorLevel:
		return "error"
	case WarnLevel:
		return "warn"
	case InfoLevel:
		return "info"
	case DebugLevel:
		return "debug"
	default:
		return fmt.Sprintf("level(%d)", int(l))
	}
}

// ParseLevel parses a string into a Level. Accepts case-insensitive prefixes.
func ParseLevel(s string) (Level, error) {
	normalized := strings.TrimSpace(strings.ToLower(s))
	switch normalized {
	case "", "info":
		return InfoLevel, nil
	case "error", "err":
		return ErrorLevel, nil
	case "warn", "warning":
		return WarnLevel, nil
	case "debug", "dbg":
		return DebugLevel, nil
	default:
		return InfoLevel, errors.New("unknown log level: " + s)
	}
}

// SimpleLogger is a lightweight structured logger (no external deps).
type SimpleLogger struct {
	mu    sync.Mutex
	lvl   Level
	out   *log.Logger
	clock func() time.Time
}

// NewSimple creates a new SimpleLogger writing to stdout with given level.
func NewSimple(l Level) *SimpleLogger {
	return &SimpleLogger{
		lvl:   l,
		out:   log.New(os.Stdout, "", 0),
		clock: time.Now,
	}
}

// WithWriter allows redirecting output (used in tests).
func (s *SimpleLogger) WithWriter(w io.Writer) *SimpleLogger {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.out = log.New(w, "", 0)
	return s
}

func (s *SimpleLogger) log(level Level, tag, msg string, kv []interface{}) {
	if level > s.lvl {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ts := s.clock().Format(time.RFC3339Nano)
	if len(kv)%2 != 0 { // ensure even pairs
		kv = append(kv, "_odd")
	}
	pairs := ""
	for i := 0; i < len(kv); i += 2 {
		k := fmt.Sprint(kv[i])
		v := fmt.Sprint(kv[i+1])
		pairs += fmt.Sprintf(" %s=%s", k, v)
	}
	s.out.Printf("%s [%s] %s%s", ts, tag, msg, pairs)
}

func (s *SimpleLogger) Info(msg string, kv ...interface{})  { s.log(InfoLevel, "INFO", msg, kv) }
func (s *SimpleLogger) Warn(msg string, kv ...interface{})  { s.log(WarnLevel, "WARN", msg, kv) }
func (s *SimpleLogger) Error(msg string, kv ...interface{}) { s.log(ErrorLevel, "ERROR", msg, kv) }
func (s *SimpleLogger) Debug(msg string, kv ...interface{}) { s.log(DebugLevel, "DEBUG", msg, kv) }

// SetLevel changes the logger verbosity at runtime.
func (s *SimpleLogger) SetLevel(l Level) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lvl = l
}

var (
	globalMu     sync.RWMutex
	globalLogger Logger = NewSimple(InfoLevel)
)

// SetGlobal sets the process-wide logger.
func SetGlobal(l Logger) {
	globalMu.Lock()
	globalLogger = l
	globalMu.Unlock()
}

// Global returns the process-wide logger.
func Global() Logger {
	globalMu.RLock()
	l := globalLogger
	globalMu.RUnlock()
	return l
}

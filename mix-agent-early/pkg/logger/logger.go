package logger

import (
	"fmt"
	"os"
	"time"
)

type Logger struct {
	prefix string
}

func New(prefix string) *Logger {
	return &Logger{prefix: prefix}
}

func (l *Logger) log(level, format string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("[%s] [%s] %s: %s\n", timestamp, level, l.prefix, msg)
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.log("INFO", format, args...)
}

func (l *Logger) Warn(format string, args ...interface{}) {
	l.log("WARN", format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.log("ERROR", format, args...)
	// Also write to kernel log if available
	if f, err := os.OpenFile("/dev/kmsg", os.O_WRONLY, 0); err == nil {
		msg := fmt.Sprintf(format, args...)
		fmt.Fprintf(f, "<%d>%s: %s\n", 3, l.prefix, msg) // 3 = LOG_ERR
		f.Close()
	}
}

func (l *Logger) Debug(format string, args ...interface{}) {
	// Only log if DEBUG env is set
	if os.Getenv("DEBUG") != "" {
		l.log("DEBUG", format, args...)
	}
}

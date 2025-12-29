package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

// Logger provides structured logging
type Logger struct {
	logger *log.Logger
}

// New creates a new logger instance
func New() *Logger {
	return &Logger{
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

// Info logs an informational message
func (l *Logger) Info(message string, fields map[string]interface{}) {
	l.log("INFO", message, fields)
}

// Error logs an error message
func (l *Logger) Error(message string, fields map[string]interface{}) {
	l.log("ERROR", message, fields)
}

// Warn logs a warning message
func (l *Logger) Warn(message string, fields map[string]interface{}) {
	l.log("WARN", message, fields)
}

func (l *Logger) log(level, message string, fields map[string]interface{}) {
	output := level + " " + message
	for k, v := range fields {
		output += " " + k + "=" + formatValue(v)
	}
	l.logger.Println(output)
}

func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case uint:
		return fmt.Sprintf("%d", val)
	case uint64:
		return fmt.Sprintf("%d", val)
	case float64:
		return fmt.Sprintf("%f", val)
	case time.Duration:
		return val.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

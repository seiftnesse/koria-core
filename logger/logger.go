package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

// Level представляет уровень логирования
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String возвращает строковое представление уровня
func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// ColorCode возвращает ANSI код цвета для уровня
func (l Level) ColorCode() string {
	switch l {
	case DEBUG:
		return "\033[36m" // Cyan
	case INFO:
		return "\033[32m" // Green
	case WARN:
		return "\033[33m" // Yellow
	case ERROR:
		return "\033[31m" // Red
	case FATAL:
		return "\033[35m" // Magenta
	default:
		return "\033[0m" // Reset
	}
}

// Logger представляет логгер с уровнями
type Logger struct {
	level      Level
	output     io.Writer
	prefix     string
	useColors  bool
	mu         sync.Mutex
	infoLog    *log.Logger
	warnLog    *log.Logger
	errorLog   *log.Logger
	debugLog   *log.Logger
	fatalLog   *log.Logger
}

// New создает новый логгер
func New(level Level, output io.Writer, prefix string) *Logger {
	if output == nil {
		output = os.Stdout
	}

	l := &Logger{
		level:     level,
		output:    output,
		prefix:    prefix,
		useColors: isTerminal(output),
	}

	flags := log.Ldate | log.Ltime | log.Lmicroseconds
	l.debugLog = log.New(output, l.formatPrefix(DEBUG), flags)
	l.infoLog = log.New(output, l.formatPrefix(INFO), flags)
	l.warnLog = log.New(output, l.formatPrefix(WARN), flags)
	l.errorLog = log.New(output, l.formatPrefix(ERROR), flags)
	l.fatalLog = log.New(output, l.formatPrefix(FATAL), flags)

	return l
}

// formatPrefix форматирует префикс с уровнем и цветом
func (l *Logger) formatPrefix(level Level) string {
	if l.useColors {
		reset := "\033[0m"
		if l.prefix != "" {
			return fmt.Sprintf("%s[%s]%s [%s] ", level.ColorCode(), level.String(), reset, l.prefix)
		}
		return fmt.Sprintf("%s[%s]%s ", level.ColorCode(), level.String(), reset)
	}

	if l.prefix != "" {
		return fmt.Sprintf("[%s] [%s] ", level.String(), l.prefix)
	}
	return fmt.Sprintf("[%s] ", level.String())
}

// isTerminal проверяет, является ли output терминалом
func isTerminal(w io.Writer) bool {
	if w == os.Stdout || w == os.Stderr {
		return true
	}
	return false
}

// SetLevel устанавливает минимальный уровень логирования
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel возвращает текущий уровень логирования
func (l *Logger) GetLevel() Level {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.level
}

// Debug логирует сообщение уровня DEBUG
func (l *Logger) Debug(format string, v ...interface{}) {
	if l.level <= DEBUG {
		l.debugLog.Printf(format, v...)
	}
}

// Info логирует сообщение уровня INFO
func (l *Logger) Info(format string, v ...interface{}) {
	if l.level <= INFO {
		l.infoLog.Printf(format, v...)
	}
}

// Warn логирует сообщение уровня WARN
func (l *Logger) Warn(format string, v ...interface{}) {
	if l.level <= WARN {
		l.warnLog.Printf(format, v...)
	}
}

// Error логирует сообщение уровня ERROR
func (l *Logger) Error(format string, v ...interface{}) {
	if l.level <= ERROR {
		l.errorLog.Printf(format, v...)
	}
}

// Fatal логирует сообщение уровня FATAL и завершает программу
func (l *Logger) Fatal(format string, v ...interface{}) {
	l.fatalLog.Printf(format, v...)
	os.Exit(1)
}

// ParseLevel парсит строку в Level
func ParseLevel(s string) (Level, error) {
	switch s {
	case "debug", "DEBUG":
		return DEBUG, nil
	case "info", "INFO":
		return INFO, nil
	case "warn", "WARN", "warning", "WARNING":
		return WARN, nil
	case "error", "ERROR":
		return ERROR, nil
	case "fatal", "FATAL":
		return FATAL, nil
	default:
		return INFO, fmt.Errorf("unknown log level: %s", s)
	}
}

// Глобальный логгер по умолчанию
var globalLogger = New(INFO, os.Stdout, "")

// SetGlobalLevel устанавливает уровень глобального логгера
func SetGlobalLevel(level Level) {
	globalLogger.SetLevel(level)
}

// SetGlobalLevelFromString устанавливает уровень глобального логгера из строки
func SetGlobalLevelFromString(s string) error {
	level, err := ParseLevel(s)
	if err != nil {
		return err
	}
	globalLogger.SetLevel(level)
	return nil
}

// Debug логирует через глобальный логгер
func Debug(format string, v ...interface{}) {
	globalLogger.Debug(format, v...)
}

// Info логирует через глобальный логгер
func Info(format string, v ...interface{}) {
	globalLogger.Info(format, v...)
}

// Warn логирует через глобальный логгер
func Warn(format string, v ...interface{}) {
	globalLogger.Warn(format, v...)
}

// Error логирует через глобальный логгер
func Error(format string, v ...interface{}) {
	globalLogger.Error(format, v...)
}

// Fatal логирует через глобальный логгер и завершает программу
func Fatal(format string, v ...interface{}) {
	globalLogger.Fatal(format, v...)
}

// Global возвращает глобальный логгер
func Global() *Logger {
	return globalLogger
}

package logger

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Config struct {
	ServiceName          string
	EnableConsoleLogging bool
	EnableFileLogging    bool
	FileBasePath         string
	LogLevel             zapcore.Level
	EnableStreamLogging  bool
	StreamHost           string
	StreamPort           string
}

var rootLogger *zap.Logger

func init() {
	var err error
	rootLogger, err = createRootLogger()

	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}

	rootLogger.Info("Logger initialized successfully")
}

func newConfigFromEnv() *Config {
	parseBool := func(key string) bool {
		v := os.Getenv(key)
		b, err := strconv.ParseBool(v)
		if err != nil {
			return false
		}
		return b
	}

	level := zap.InfoLevel
	if lvlStr := os.Getenv("LOGGING_LEVEL"); lvlStr != "" {
		_ = level.UnmarshalText([]byte(lvlStr))
	}

	return &Config{
		ServiceName:          os.Getenv("SERVICE_NAME"),
		EnableConsoleLogging: parseBool("LOGGING_CONSOLE_ENABLED"),
		EnableFileLogging:    parseBool("LOGGING_FILE_ENABLED"),
		FileBasePath:         os.Getenv("LOGGING_FILE_BASE_PATH"),
		LogLevel:             level,
		EnableStreamLogging:  parseBool("LOGGING_STREAM_ENABLED"),
		StreamHost:           os.Getenv("LOGGING_STREAM_HOST"),
		StreamPort:           os.Getenv("LOGGING_STREAM_PORT"),
	}
}

func createRootLogger() (*zap.Logger, error) {
	var cores []zapcore.Core

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:       "timestamp",
		LevelKey:      "level",
		NameKey:       "logger",
		MessageKey:    "message",
		StacktraceKey: "stackTrace",
		CallerKey:     "caller",
		EncodeTime:    zapcore.ISO8601TimeEncoder,
		EncodeLevel:   zapcore.CapitalLevelEncoder,
		EncodeCaller:  zapcore.ShortCallerEncoder,
	}

	encoderConfig.EncodeTime = zapcore.TimeEncoder(func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.UTC().Format("2006-01-02T15:04:05.000Z"))
	})

	cfg := newConfigFromEnv()

	jsonEncoder := zapcore.NewJSONEncoder(encoderConfig)
	levelEnabler := zap.NewAtomicLevelAt(cfg.LogLevel)

	if cfg.EnableConsoleLogging {
		consoleCore := zapcore.NewCore(jsonEncoder, zapcore.AddSync(os.Stdout), levelEnabler)
		cores = append(cores, consoleCore)
	}

	if cfg.EnableFileLogging {
		logFilePath := cfg.FileBasePath + "/" + cfg.ServiceName + ".log"
		ljLogger := &lumberjack.Logger{
			Filename:   logFilePath,
			MaxSize:    100,
			MaxBackups: 7,
			MaxAge:     30,
			Compress:   true,
		}
		fileCore := zapcore.NewCore(jsonEncoder, zapcore.AddSync(ljLogger), levelEnabler)
		cores = append(cores, fileCore)
	}

	if cfg.EnableStreamLogging {
		tcpWriter, err := NewTCPAsyncWriter(cfg.StreamHost, cfg.StreamPort)
		if err != nil {
			return nil, err
		}
		streamCore := zapcore.NewCore(jsonEncoder, zapcore.AddSync(tcpWriter), levelEnabler)
		cores = append(cores, streamCore)
	}

	combinedCore := zapcore.NewTee(cores...)

	rootLogger = zap.New(combinedCore, zap.AddCaller(), zap.AddCallerSkip(1)).
		With(
			zap.String("service", cfg.ServiceName),
		)
	return rootLogger, nil
}

type TCPWriter struct {
	conn net.Conn
}

func NewTCPWriter(host string, port string) (*TCPWriter, error) {
	address := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return nil, err
	}
	return &TCPWriter{conn: conn}, nil
}

func (w *TCPWriter) Write(p []byte) (n int, err error) {
	return w.conn.Write(p)
}

// Debug logs a debug-level message.
func Debug(msg string, fields ...zap.Field) {
	rootLogger.Debug(msg, fields...)
}

// Info logs an info-level message.
func Info(msg string, fields ...zap.Field) {
	rootLogger.Info(msg, fields...)
}

// Warn logs a warn-level message.
func Warn(msg string, fields ...zap.Field) {
	rootLogger.Warn(msg, fields...)
}

// Error logs an error-level message.
func Error(msg string, fields ...zap.Field) {
	rootLogger.Error(msg, fields...)
}

// DPanic logs a DPanic-level message.
func DPanic(msg string, fields ...zap.Field) {
	rootLogger.DPanic(msg, fields...)
}

// Panic logs a panic-level message and then panics.
func Panic(msg string, fields ...zap.Field) {
	rootLogger.Panic(msg, fields...)
}

// Fatal logs a fatal-level message and then exits the application.
func Fatal(msg string, fields ...zap.Field) {
	rootLogger.Fatal(msg, fields...)
}

// IsDebugEnabled returns true if the Debug level is enabled.
func IsDebugEnabled() bool {
	return rootLogger.Core().Enabled(zap.DebugLevel)
}

// IsInfoEnabled returns true if the Info level is enabled.
func IsInfoEnabled() bool {
	return rootLogger.Core().Enabled(zap.InfoLevel)
}

// IsWarnEnabled returns true if the Warn level is enabled.
func IsWarnEnabled() bool {
	return rootLogger.Core().Enabled(zap.WarnLevel)
}

// IsErrorEnabled returns true if the Error level is enabled.
func IsErrorEnabled() bool {
	return rootLogger.Core().Enabled(zap.ErrorLevel)
}

// IsDPanicEnabled returns true if the DPanic level is enabled.
func IsDPanicEnabled() bool {
	return rootLogger.Core().Enabled(zap.DPanicLevel)
}

// IsPanicEnabled returns true if the Panic level is enabled.
func IsPanicEnabled() bool {
	return rootLogger.Core().Enabled(zap.PanicLevel)
}

// IsFatalEnabled returns true if the Fatal level is enabled.
func IsFatalEnabled() bool {
	return rootLogger.Core().Enabled(zap.FatalLevel)
}

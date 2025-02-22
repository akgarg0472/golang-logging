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

var RootLogger *zap.Logger

func init() {
	fmt.Println("initializing logger")

	var err error
	RootLogger, err = createRootLogger()

	if err != nil {
		panic(err)
	}

	RootLogger.Info("Logger initialized successfully")
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
		tcpWriter, err := NewTCPWriter(cfg.StreamHost, cfg.StreamPort)
		if err != nil {
			return nil, err
		}
		streamCore := zapcore.NewCore(jsonEncoder, zapcore.AddSync(tcpWriter), levelEnabler)
		cores = append(cores, streamCore)
	}

	combinedCore := zapcore.NewTee(cores...)

	RootLogger = zap.New(combinedCore, zap.AddCaller(), zap.AddCallerSkip(2)).
		With(
			zap.String("service", cfg.ServiceName),
		)
	return RootLogger, nil
}

type TCPWriter struct {
	conn net.Conn
}

func NewTCPWriter(host, port string) (*TCPWriter, error) {
	address := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return nil, err
	}
	return &TCPWriter{conn: conn}, nil
}

func (w *TCPWriter) Write(p []byte) (n int, err error) {
	newP := append([]byte{}, p...)
	newP = append(newP, '\n')
	return w.conn.Write(newP)
}

package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	auth "github.com/akgarg0472/golang-logging/internal"
	"github.com/akgarg0472/golang-logging/logger"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func init() {
	fmt.Println("initializing main")
}

func main() {
	log := logger.RootLogger
	defer log.Sync()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(LoggingMiddleware(log))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		reqLogger := log.With(zap.String("requestId", middleware.GetReqID(r.Context())))
		auth.Login("admin")
		reqLogger.Info("handling root endpoint")
		w.Write([]byte("Hello, golang-logging!"))
	})

	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "0"
	}

	listenAddr := ":" + serverPort
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatal("Error starting listener", zap.Error(err))
	}

	actualAddr := listener.Addr().String()
	host, port, err := net.SplitHostPort(actualAddr)
	if err != nil {
		log.Warn("Unable to split host and port", zap.String("addr", actualAddr))
		host = actualAddr
		port = serverPort
	}

	log = log.With(zap.String("host", host), zap.String("port", port))

	log.Info("Server is starting", zap.String("actualAddr", actualAddr))

	if err := http.Serve(listener, r); err != nil {
		log.Fatal("HTTP server failed", zap.Error(err))
	}
}

func LoggingMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := middleware.GetReqID(r.Context())
			reqLogger := logger.With(zap.String("requestId", requestID))
			next.ServeHTTP(w, r)
			reqLogger.Info("Request completed", zap.String("method", r.Method), zap.String("url", strings.Split(r.URL.String(), "?")[0]))
		})
	}
}

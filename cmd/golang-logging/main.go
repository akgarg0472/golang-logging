package main

import (
	"net"
	"net/http"
	"os"

	internal "github.com/akgarg0472/golang-logging/internal"
	log "github.com/akgarg0472/golang-logging/logger"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func main() {

	r := chi.NewRouter()
	r.Use(middleware.RequestID)

	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		log.Info("handling root endpoint")
		internal.Login("requestId", "root")
		w.Header().Add("Content-Type", "application/json")
		w.Write([]byte("{'message': 'PONG'}"))
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

	log.Info("Server Host and Port", zap.String("host", host), zap.String("port", port))

	log.Info("Server is starting", zap.String("actualAddr", actualAddr))

	if err := http.Serve(listener, r); err != nil {
		log.Fatal("HTTP server failed", zap.Error(err))
	}
}

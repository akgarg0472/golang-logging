package internal

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/akgarg0472/golang-logging/logger"
)

var log *zap.Logger

func init() {
	fmt.Println("initializing authService")
	log = logger.RootLogger
}

func Login(username string) bool {
	log.Info("Login user", zap.String("username", username))
	return username == "root"
}

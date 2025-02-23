package internal

import (
	"go.uber.org/zap"

	log "github.com/akgarg0472/golang-logging/logger"
)

func Login(requestID string, username string) bool {
	if log.IsDebugEnabled() {
		log.Debug("Login user", zap.String("username", username), zap.Any("requestId", requestID))
	}
	return username == "root"
}

/*
 * Marathon API
 *
 * One login for all your fitness data needs.
 *
 * API version: 0.1.0
 */
package service

import (
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

func SetupLogger() *logrus.Logger {
	logger := logrus.New()

	// Create file to store logs
	file, err := os.OpenFile("runtime.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		logger.Out = file
	} else {
		logger.Info("Failed to log to file, using default stderr")
	}

	// Log formatting
	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})
	logger.SetLevel(logrus.DebugLevel)

	return logger
}

func Logger(logger *logrus.Logger, next http.Handler, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		logger.WithFields(logrus.Fields{
			"method": r.Method,
			"uri":    r.RequestURI,
			"func":   name,
			"took":   time.Since(start),
		}).Info("Request received")
	})
}

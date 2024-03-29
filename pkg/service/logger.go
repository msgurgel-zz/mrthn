/*
 * mrthn API
 *
 * One login for all your fitness data needs.
 *
 * API version: 0.1.0
 */
package service

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

func SetupLogger(logToStderr bool) *logrus.Logger {
	logger := logrus.New()

	if !logToStderr {
		// Create file to store logs
		logDir := filepath.Join(".", "log")
		_ = os.MkdirAll(logDir, 0700)
		file, err := os.OpenFile("log/mrthn.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			logger.Out = file
		} else {
			logger.Infof("Failed to log to file, using default stderr - err: %s", err)
		}
	}

	// Log formatting
	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})
	logger.SetLevel(logrus.DebugLevel)

	return logger
}

func Logger(log *logrus.Logger, next http.Handler, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		log.WithFields(logrus.Fields{
			"method": r.Method,
			"uri":    r.RequestURI,
			"func":   name,
			"took":   time.Since(start),
		}).Info("Request received")
	})
}

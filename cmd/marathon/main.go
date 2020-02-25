/*
 * Marathon API
 *
 * One login for all your fitness data needs.
 *
 * API version: 0.1.0
 */
package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/msgurgel/marathon/pkg/dal"

	"github.com/msgurgel/marathon/pkg/platform"

	"github.com/sirupsen/logrus"

	"github.com/msgurgel/marathon/pkg/environment"

	"github.com/msgurgel/marathon/pkg/service"
)

func main() {
	var wait time.Duration
	flag.DurationVar(
		&wait,
		"graceful-timeout",
		time.Second*15,
		"the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m",
	)

	var logToStderr bool
	flag.BoolVar(
		&logToStderr,
		"log-to-stderr",
		false,
		"log to stderr instead of file",
	)

	flag.Parse()

	log := service.SetupLogger(logToStderr)

	// get the environment variables
	env, err := environment.ReadEnvFile()
	if err != nil {
		log.WithFields(logrus.Fields{
			"err": err,
		}).Fatal("failed to read .env file. Exiting...")
	}

	// Connect to database using configuration struct
	db, err := dal.InitializeDBConn(
		env.Database.Host,
		env.Database.Port,
		env.Database.User,
		env.Database.Password,
		env.Database.DatabaseName,
	)
	if err != nil {
		log.WithFields(logrus.Fields{
			"err": err,
		}).Fatal("failed to connect to database. Exiting...")
	}

	defer db.Close()

	// Setup connections to platforms
	platform.InitializePlatforms(db, log)

	router := service.NewRouter(db, log, env)

	srv := &http.Server{
		Addr:         env.Server.Address,
		Handler:      router,
		ReadTimeout:  env.Server.ReadTimeOut,
		WriteTimeout: env.Server.IdleTimeout,
		IdleTimeout:  env.Server.IdleTimeout,
	}

	// Run server in a goroutine so that it doesn't block
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Info(err)
		}
	}()

	log.WithFields(logrus.Fields{
		"addr": env.Server.Address,
	}).Info("Server started")

	c := make(chan os.Signal, 1)

	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()

	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	_ = srv.Shutdown(ctx)

	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Info("Shutting down...")
	os.Exit(0)
}

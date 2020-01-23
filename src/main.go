package main

/*
	File: main.go
	Last Updated: January 23, 2020
	Updated by: Manuel Poppe Richter
	Purpose: This is the main server file, where it creates our web service, and starts listening at its' endpoints for requests
	This page also has a lot of fluff from the tutorial (extra channels, etc) which me may be able to get rid of
*/

import (
	"context"
	"flag"
	"fmt"
	"github.com/msgurgel/marathon/pkg/service"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var (
		httpAddr = flag.String("http", ":8080", "http listen address")
	)
	flag.Parse()
	ctx := context.Background()

	// creating instance of the service
	srv := service.MarathonWebService{}
	errChan := make(chan error)

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errChan <- fmt.Errorf("%s", <-c)
	}()

	// mapping endpoints
	endpoints := service.Endpoints{
		GetCaloriesEndpoint: service.MakeGetCaloriesEndpoint(srv),
		GetStepsEndpoint:    service.MakeGetStepsEndpoint(srv),
	}

	// HTTP transport
	go func() {
		log.Println("service is listening on port:", *httpAddr)
		handler := service.NewHTTPServer(ctx, endpoints)
		errChan <- http.ListenAndServe(*httpAddr, handler)
	}()

	log.Fatalln(<-errChan)
}

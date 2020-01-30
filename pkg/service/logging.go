package service

/*
	File:    logging.go
	Date:    January 30, 2020
	Author:  Mateus Gurgel
	Purpose: Logging middleware that logs requests to the service's endpoints
*/

import (
    "context"
    "github.com/sirupsen/logrus"
    "time"
)

// LoggingMiddleware is a logrus logger that is used as a middleware for requests
type LoggingMiddleware struct {
    Logger *logrus.Logger
    Next MarathonService
}

// GetCalories acts as a middleware for the method of the same name from MarathonWebService
func (mw LoggingMiddleware) GetCalories(ctx context.Context, userId int) (output int, err error) {
    defer func(begin time.Time) {
        mw.Logger.WithFields(logrus.Fields{
            "method": "GetCalories",
            "input": userId,
            "output": output,
            "err": err,
            "took": time.Since(begin),
        }).Info("request received")
    }(time.Now())

    output,err = mw.Next.GetCalories(ctx, userId)
    return
}

// GetSteps acts as a middleware for the method of the same name from MarathonWebService
func (mw LoggingMiddleware) GetSteps(ctx context.Context, userId int) (output int, err error) {
    defer func(begin time.Time) {
        mw.Logger.WithFields(logrus.Fields{
            "method": "GetSteps",
            "input": userId,
            "output": output,
            "err": err,
            "took": time.Since(begin),
        }).Info("request received")
    }(time.Now())

    output,err = mw.Next.GetSteps(ctx, userId)
    return
}

package platform

import (
	"database/sql"
	"time"

	"github.com/sirupsen/logrus"
)

var Platforms map[string]Platform

type Platform interface {
	Name() string
	GetSteps(user int, date time.Time) (int, error)
	GetCalories(user int, date time.Time) (int, error)
}

func InitializePlatforms(db *sql.DB, log *logrus.Logger) {
	Platforms = make(map[string]Platform)

	Platforms["fitbit"] = Fitbit{db: db, log: log}
}

func GetPlatforms(platformArr []string) []Platform {

	// TODO: deal with panic in case of str no being in the map
	var results []Platform
	for _, platform := range platformArr {
		results = append(results, Platforms[platform])
	}

	return results
}

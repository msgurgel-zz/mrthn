package platform

import (
	"database/sql"
	"time"

	"github.com/msgurgel/marathon/pkg/auth"

	"github.com/msgurgel/marathon/pkg/dal"

	"github.com/sirupsen/logrus"
)

var Platforms map[string]Platform

type Platform interface {
	Name() string
	GetSteps(user int, date time.Time) (int, error)
	GetCalories(user int, date time.Time) (int, error)
	GetDistance(user int, date time.Time) (float64, error)
}

func InitializePlatforms(db *sql.DB, log *logrus.Logger, authTypes auth.Types) {
	domains, err := dal.GetPlatformDomains(db)
	if err != nil {
		log.WithFields(logrus.Fields{
			"err": err,
		}).Fatal("unable to get domains from the db")
	}

	Platforms = make(map[string]Platform)

	Platforms["fitbit"] = Fitbit{
		db:            db,
		log:           log,
		domain:        domains["fitbit"],
		authorization: authTypes.Oauth2.Configs["fitbit"],
	}

	Platforms["google"] = Google{
		db:            db,
		log:           log,
		domain:        domains["google"],
		authorization: authTypes.Oauth2.Configs["google"],
	}
}

func GetPlatforms(platformNames []string) []Platform {
	// TODO: deal with panic in case of str no being in the map
	var results []Platform
	for _, platform := range platformNames {
		results = append(results, Platforms[platform])
	}

	return results
}

func IsPlatformAvailable(platform string) bool {
	for availablePlatform := range Platforms {
		if platform == availablePlatform {
			return true
		}
	}

	return false
}

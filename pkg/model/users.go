package model

import (
	"database/sql"
	"errors"
	"time"

	"github.com/msgurgel/marathon/pkg/dal"
	"github.com/msgurgel/marathon/pkg/platform"

	"github.com/sirupsen/logrus"
)

type ValueResult struct {
	Platform string `json:"platform,omitempty"`
	Value    int    `json:"value"`
}

// TODO: Can this be refactored, so there isn't as much copied code from GetUserSteps?
func GetUserCalories(db *sql.DB, log *logrus.Logger, userId int, date time.Time) ([]ValueResult, error) {
	platforms, err := getPlatforms(db, userId, log)
	if err != nil {
		return nil, err
	}

	// Request steps from each platform
	var caloriesValues []ValueResult
	for _, p := range platforms {
		result, err := p.GetCalories(userId, date)
		if err != nil {
			log.WithFields(logrus.Fields{
				"err":    err,
				"userId": userId,
				"date":   date.Format("2006-01-02"), // TODO: make this layout shared somehow?
				"plat":   p.Name(),
			}).Error("failed to call GetCalories for platform")
			continue // Try the next platform
		}

		// Format result and add to caloriesValues
		caloriesVal := ValueResult{
			Platform: p.Name(),
			Value:    result,
		}
		caloriesValues = append(caloriesValues, caloriesVal)
	}

	if len(caloriesValues) == 0 {
		return nil, errors.New("could not connect to any platforms, try again later")
	}
	return caloriesValues, nil
}

func GetUserSteps(db *sql.DB, log *logrus.Logger, userId int, date time.Time) ([]ValueResult, error) {
	platforms, err := getPlatforms(db, userId, log)
	if err != nil {
		return nil, err
	}

	// Request steps from each platform
	var stepsValues []ValueResult
	for _, p := range platforms {
		result, err := p.GetSteps(userId, date)
		if err != nil {
			log.WithFields(logrus.Fields{
				"err":    err,
				"userId": userId,
				"date":   date.Format("2006-01-02"), // TODO: make this layout shared somehow?
				"plat":   p.Name(),
			}).Error("failed to call GetSteps for platform")
			continue // Try the next platform
		}

		// Format result and add to stepsValues
		stepVal := ValueResult{
			Platform: p.Name(),
			Value:    result,
		}
		stepsValues = append(stepsValues, stepVal)
	}

	if len(stepsValues) == 0 {
		return nil, errors.New("could not connect to any platforms, try again later")
	}
	return stepsValues, nil
}

func getPlatforms(db *sql.DB, userId int, log *logrus.Logger) ([]platform.Platform, error) {
	platformStr, err := dal.GetPlatformsString(db, userId)
	if err != nil {
		log.WithFields(logrus.Fields{
			"err":    err,
			"userId": userId,
		}).Error("failed to get platforms associated to user")

		return nil, errors.New("server error, try again later")
	}

	platforms := platform.GetPlatforms(platformStr)
	return platforms, nil
}

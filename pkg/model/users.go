package model

import (
	"database/sql"
	"errors"
	"time"

	"github.com/msgurgel/mrthn/pkg/helpers"

	"github.com/msgurgel/mrthn/pkg/dal"
	"github.com/msgurgel/mrthn/pkg/platform"

	"github.com/sirupsen/logrus"
)

type ValueResult struct {
	Platform string  `json:"platform,omitempty"`
	Value    float64 `json:"value"`
}

type GetValueParams struct {
	DB          *sql.DB
	Log         *logrus.Logger
	UserID      int
	Date        time.Time
	LargestOnly bool
}

// TODO: Can this be refactored, so there isn't as much copied code from GetUserSteps?
func GetUserCalories(params GetValueParams) ([]ValueResult, error) {
	platforms, err := getPlatforms(params.DB, params.UserID, params.Log)
	if err != nil {
		return nil, err
	}

	// Request steps from each platform
	var caloriesValues []ValueResult
	for _, p := range platforms {
		result, err := p.GetCalories(params.UserID, params.Date)
		if err != nil {
			params.Log.WithFields(logrus.Fields{
				"err":    err,
				"userID": params.UserID,
				"date":   params.Date.Format(helpers.ISOLayout),
				"plat":   p.Name(),
			}).Error("failed to call GetCalories for platform")
			continue // Try the next platform
		}

		// Format result and add to caloriesValues
		caloriesVal := ValueResult{
			Platform: p.Name(),
			Value:    float64(result),
		}
		caloriesValues = append(caloriesValues, caloriesVal)
	}

	if len(caloriesValues) == 0 {
		return nil, errors.New("could not connect to any platforms, try again later")
	}

	// If the user only wants the largest amount, filter out the other results
	if params.LargestOnly {
		return filterNonLargest(caloriesValues), nil
	}

	return caloriesValues, nil
}

func GetUserSteps(params GetValueParams) ([]ValueResult, error) {
	platforms, err := getPlatforms(params.DB, params.UserID, params.Log)
	if err != nil {
		return nil, err
	}

	// Request steps from each platform
	var stepsValues []ValueResult
	for _, p := range platforms {
		result, err := p.GetSteps(params.UserID, params.Date)
		if err != nil {
			params.Log.WithFields(logrus.Fields{
				"err":    err,
				"userID": params.UserID,
				"date":   params.Date.Format(helpers.ISOLayout),
				"plat":   p.Name(),
			}).Error("failed to call GetSteps for platform")
			continue // Try the next platform
		}

		// Format result and add to stepsValues
		stepVal := ValueResult{
			Platform: p.Name(),
			Value:    float64(result),
		}
		stepsValues = append(stepsValues, stepVal)
	}

	if len(stepsValues) == 0 {
		return nil, errors.New("could not connect to any platforms, try again later")
	}

	// If the user only wants the largest amount, filter out the other results
	if params.LargestOnly {
		return filterNonLargest(stepsValues), nil
	}

	return stepsValues, nil
}

func GetUserDistance(params GetValueParams) ([]ValueResult, error) {
	platforms, err := getPlatforms(params.DB, params.UserID, params.Log)
	if err != nil {
		return nil, err
	}

	// Request steps from each platform
	var distanceValues []ValueResult
	for _, p := range platforms {
		result, err := p.GetDistance(params.UserID, params.Date)
		if err != nil {
			params.Log.WithFields(logrus.Fields{
				"err":    err,
				"userID": params.UserID,
				"date":   params.Date.Format(helpers.ISOLayout),
				"plat":   p.Name(),
			}).Error("failed to call GetDistance for platform")
			continue // Try the next platform
		}

		// Format result and add to stepsValues
		distanceVal := ValueResult{
			Platform: p.Name(),
			Value:    result,
		}
		distanceValues = append(distanceValues, distanceVal)
	}

	if len(distanceValues) == 0 {
		return nil, errors.New("could not connect to any platforms, try again later")
	}

	// If the user only wants the largest amount, filter out the other results
	if params.LargestOnly {
		return filterNonLargest(distanceValues), nil
	}

	return distanceValues, nil
}

func GetUserDistanceOverPeriod(params GetValueParams, period string) ([]ValueResult, error) {
	platforms, err := getPlatforms(params.DB, params.UserID, params.Log)
	if err != nil {
		return nil, err
	}

	// Request steps from each platform
	var distanceValues []ValueResult
	for _, p := range platforms {
		result, err := p.GetDistanceOverPeriod(params.UserID, params.Date, period)
		if err != nil {
			params.Log.WithFields(logrus.Fields{
				"err":    err,
				"userID": params.UserID,
				"date":   params.Date.Format(helpers.ISOLayout),
				"plat":   p.Name(),
			}).Error("failed to call GetDistanceOverPeriod for platform")
			continue // Try the next platform
		}

		// Format result and add to distanceValues
		distanceVal := ValueResult{
			Platform: p.Name(),
			Value:    result,
		}
		distanceValues = append(distanceValues, distanceVal)
	}

	if len(distanceValues) == 0 {
		return nil, errors.New("could not connect to any platforms, try again later")
	}

	// If the user only wants the largest amount, filter out the other results
	if params.LargestOnly {
		return filterNonLargest(distanceValues), nil
	}

	return distanceValues, nil
}

func getPlatforms(db *sql.DB, userID int, log *logrus.Logger) ([]platform.Platform, error) {
	platformStr, err := dal.GetPlatformNames(db, userID)
	if err != nil {
		log.WithFields(logrus.Fields{
			"err":    err,
			"userID": userID,
		}).Error("failed to get platforms associated to user")

		return nil, errors.New("server error, try again later")
	}

	platforms := platform.GetPlatforms(platformStr)
	return platforms, nil
}

func filterNonLargest(resultValues []ValueResult) []ValueResult {
	if len(resultValues) <= 1 {
		return resultValues
	}

	var maxValue float64 = 0
	var maxIndex = 0

	for index, caloriesValue := range resultValues {
		if caloriesValue.Value >= maxValue {
			maxIndex = index
			maxValue = caloriesValue.Value
		}
	}

	var maxCaloriesValue []ValueResult
	maxCaloriesValue = append(maxCaloriesValue, resultValues[maxIndex])

	return maxCaloriesValue
}

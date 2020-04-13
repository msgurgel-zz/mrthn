package scheduled

import (
	"database/sql"
	"time"

	"github.com/msgurgel/marathon/pkg/model"

	"github.com/msgurgel/marathon/pkg/dal"

	"github.com/sirupsen/logrus"
)

const calorieType = 1
const stepType = 2
const distanceType = 3

func UpdateUserData(db *sql.DB, log *logrus.Logger) error {
	// First, we need to get the list of all users who have signed up
	userIDs, err := dal.GetAllUserIdsInCredentials(db)

	if err != nil {
		return err
	}

	// Get today's date
	currentTime := time.Now()
	currentDate := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, time.UTC)

	for _, currentID := range userIDs {

		currentUserData := dal.UserData{
			Platforms: map[string]map[string]float64{},
			UserID:    currentID,
			Date:      currentDate,
		}

		currentUserData.Platforms["fitbit"] = map[string]float64{}
		currentUserData.Platforms["google"] = map[string]float64{}
		currentUserData.Platforms["strava"] = map[string]float64{}

		userParams := model.GetValueParams{
			DB:          db,
			Log:         log,
			UserID:      currentID,
			Date:        currentDate,
			LargestOnly: false,
		}

		// We need to get the calorie, steps and distance amount for each userID
		response, err := model.GetUserCalories(userParams)

		if err != nil {
			log.WithFields(logrus.Fields{
				"error": err,
			}).Error("error occurred while attempting to get user calories")
			return err
		}

		addUserData(&currentUserData, response, "calories")

		response, err = model.GetUserSteps(userParams)

		if err != nil {
			log.WithFields(logrus.Fields{
				"error": err,
			}).Error("error occurred while attempting to get user steps")
			return err
		}

		addUserData(&currentUserData, response, "steps")

		response, err = model.GetUserDistance(userParams)

		if err != nil {
			log.WithFields(logrus.Fields{
				"error": err,
			}).Error("error occurred while attempting to get user distance")
			return err
		}

		addUserData(&currentUserData, response, "distance")

		// Now that the queries for the user has been completed, we can add the data to the user_data table
		_, err = dal.AddUserData(db, currentUserData)

		if err != nil {
			return err
		}
	}

	return nil

}

func addUserData(data *dal.UserData, result []model.ValueResult, resourceType string) {
	// For each result in the set, we have to add it to the UserData
	for _, currentResult := range result {
		// Check the platform of this result
		switch currentResult.Platform {
		case "google":
			data.Platforms["google"][resourceType] = currentResult.Value
		case "fitbit":
			data.Platforms["fitbit"][resourceType] = currentResult.Value
		case "strava":
			data.Platforms["strava"][resourceType] = currentResult.Value
		}

	}
}

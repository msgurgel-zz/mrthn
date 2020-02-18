package dal

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/DATA-DOG/go-sqlmock"
)

var DB *sql.DB
var Mock sqlmock.Sqlmock

func TestMain(m *testing.M) {
	var err error

	DB, Mock, err = sqlmock.New()
	if err != nil {
		log.Fatalf("failed while setting up mock db: %s", err.Error())
	}

	code := m.Run()
	DB.Close()

	os.Exit(code)
}

func TestGetUserTokensHappyPath(t *testing.T) {
	platform := "fitbit"
	userId := 1

	cols := []string{
		"access_token",
		"refresh_token",
	}
	rows := sqlmock.NewRows(cols).AddRow("AC3SST0K3N", "R3FR3SHT0K3N")

	expectedSQL := fmt.Sprintf("^SELECT (.+) FROM oauth2 o JOIN %q (.+) WHERE user_id = %d*", platform, userId)
	Mock.ExpectQuery(expectedSQL).WillReturnRows(rows)

	accessTkn, refreshTkn, err := GetUserTokens(DB, userId, platform)
	if err != nil {
		t.Errorf("failed to get user tokens: %s", err.Error())
		return
	}

	assert.Equal(t, "AC3SST0K3N", accessTkn)
	assert.Equal(t, "R3FR3SHT0K3N", refreshTkn)
}

func TestGetPlatformsString(t *testing.T) {
	userId := 1
	expectedPlatforms := "fitbit,garmin,google-fit,map-my-tracks"

	cols := []string{
		"platforms",
	}
	rows := sqlmock.NewRows(cols).AddRow(expectedPlatforms)

	expectedSQL := fmt.Sprintf(`^SELECT platforms FROM "user" WHERE id = %d*`, userId)
	Mock.ExpectQuery(expectedSQL).WillReturnRows(rows)

	platformStr, err := GetPlatformsString(DB, userId)
	if err != nil {
		t.Errorf("failed to get platforms: %s", err.Error())
		return
	}

	assert.Equal(t, expectedPlatforms, platformStr)
}

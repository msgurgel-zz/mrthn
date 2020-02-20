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
	platformName := "fitbit"
	userId := 1

	cols := []string{
		"connection_string",
	}
	rows := sqlmock.NewRows(cols).AddRow("oauth2;AC3SST0K3N;R3FR3SHT0K3N")

	expectedSQL := fmt.Sprintf("^SELECT connection_string FROM credentials WHERE user_id = %d AND platform_name = %q*", userId, platformName)
	Mock.ExpectQuery(expectedSQL).WillReturnRows(rows)

	accessTkn, refreshTkn, err := GetUserTokens(DB, userId, platformName)
	if err != nil {
		t.Errorf("failed to get user tokens: %s", err.Error())
		return
	}

	assert.Equal(t, "AC3SST0K3N", accessTkn)
	assert.Equal(t, "R3FR3SHT0K3N", refreshTkn)
}

func TestGetPlatformNames(t *testing.T) {
	userId := 1
	expectedPlatforms := []string{"fitbit", "garmin", "google-fit", "map-my-tracks"}

	cols := []string{
		"platform_name",
	}

	rows := sqlmock.NewRows(cols)
	for _, platName := range expectedPlatforms {
		rows = rows.AddRow(platName)
	}

	expectedSQL := fmt.Sprintf(`^SELECT platform_name FROM "credentials" WHERE user_id = %d*`, userId)
	Mock.ExpectQuery(expectedSQL).WillReturnRows(rows)

	platformStr, err := GetPlatformNames(DB, userId)
	if err != nil {
		t.Errorf("failed to get platforms: %s", err.Error())
		return
	}

	assert.Equal(t, expectedPlatforms, platformStr)
}

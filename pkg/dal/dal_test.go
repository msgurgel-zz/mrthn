package dal

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"

	"golang.org/x/crypto/bcrypt"

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

func TestInsertSecretInExistingClient_ShouldInsertSecret(t *testing.T) {
	// Prepare params and expected results
	secret := []byte("my_secret")
	clientID := 1

	// Mock expected SQL queries
	Mock.ExpectExec(`^UPDATE client SET secret`).
		WithArgs(secret, clientID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Call the func that we are testing
	rowsAffected, err := InsertSecretInExistingClient(DB, clientID, secret)

	// Assertions
	if err != nil {
		t.Errorf("error was not expected when inserting secret: %s", err)
	}

	// We make sure that all expectations were met
	if err := Mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t, int64(1), rowsAffected)
}

func TestGetClientSecret_ShouldGetSecret(t *testing.T) {
	// Prepare params and expected results
	clientID := 1
	secret := []byte("my_secret")

	// Mock SQL rows
	cols := []string{
		"secret",
	}
	rows := sqlmock.NewRows(cols).AddRow(secret)

	// Mock expected SQL queries
	expectedSQL := fmt.Sprintf("^SELECT secret FROM client WHERE id = %d$", clientID)
	Mock.ExpectQuery(expectedSQL).WillReturnRows(rows)

	// Call the func that we are testing
	actualSecret, err := GetClientSecret(DB, clientID)

	// Assertions
	if err != nil {
		t.Errorf("error was not expected when getting secret: %s", err)
	}

	// We make sure that all expectations were met
	if err := Mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t, secret, actualSecret)
}

func TestGetUserTokens_ShouldGetTokens(t *testing.T) {
	platformName := "fitbit"
	userID := 1
	platformID := 1

	platformIDQuery := fmt.Sprintf("^SELECT id FROM platform WHERE name = '%s'$", platformName)
	Mock.ExpectQuery(platformIDQuery).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(platformID))

	cols := []string{
		"connection_string",
	}
	rows := sqlmock.NewRows(cols).AddRow("oauth2;AC3$$T0K3N;R3FR3$HT0K3N")

	expectedSQL := fmt.Sprintf("^SELECT connection_string FROM credentials WHERE user_id = %d AND platform_id = %d$", userID, platformID)
	Mock.ExpectQuery(expectedSQL).WillReturnRows(rows)

	accessTkn, refreshTkn, err := GetUserTokens(DB, userID, platformName)
	if err != nil {
		t.Errorf("failed to get user tokens: %s", err.Error())
		return
	}

	// We make sure that all expectations were met
	if err := Mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t, "AC3$$T0K3N", accessTkn)
	assert.Equal(t, "R3FR3$HT0K3N", refreshTkn)
}

func TestGetPlatformNames(t *testing.T) {
	userID := 1
	expectedPlatforms := []string{"fitbit", "garmin", "google-fit", "map-my-tracks"}

	cols := []string{
		"name",
	}

	rows := sqlmock.NewRows(cols)
	for _, platName := range expectedPlatforms {
		rows = rows.AddRow(platName)
	}

	expectedSQL := fmt.Sprintf(`^SELECT name FROM platform p JOIN (.+) WHERE user_id = %d$`, userID)
	Mock.ExpectQuery(expectedSQL).WillReturnRows(rows)

	platformStr, err := GetPlatformNames(DB, userID)
	if err != nil {
		t.Errorf("failed to get platforms: %s", err.Error())
		return
	}

	// We make sure that all expectations were met
	if err := Mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t, expectedPlatforms, platformStr)
}

func TestGetUserByPlatformID(t *testing.T) {
	platID := "A1B2C3"
	platName := "fitbit"
	expectedUserID := 420

	cols := []string{
		"user_id",
	}

	rows := sqlmock.NewRows(cols).AddRow(expectedUserID)

	expectedSQL := fmt.Sprintf(
		"^SELECT user_id FROM credentials [a-z] "+
			"JOIN platform [a-z]+ ON (.+) "+
			"WHERE [a-z]+.name = '%s' AND [a-z]+.upid = '%s'$",
		platName,
		platID,
	)
	Mock.ExpectQuery(expectedSQL).WillReturnRows(rows)

	userID, err := GetUserByPlatformID(DB, platID, platName)
	if err != nil {
		t.Errorf("failed to get user: %s", err.Error())
		return
	}

	// We make sure that all expectations were met
	if err := Mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t, expectedUserID, userID)
}

func TestInsertUserCredentials_ShouldInsertCredentials(t *testing.T) {
	// Prepare params and expected results
	userID := 1
	clientID := 1
	platID := 1
	platName := "fitbit"
	UPID := "A1B2C3"
	connStr := "oauth2;AC3$$T0K3N;R3FR3$HT0K3N"

	// Mock expected DB calls in order
	Mock.ExpectBegin()
	Mock.ExpectQuery(
		`^INSERT INTO "user" DEFAULT VALUES RETURNING id$`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(userID))

	expectedPlatIDSQL := fmt.Sprintf(`^SELECT id FROM platform WHERE name = '%s'$`, platName)
	Mock.ExpectQuery(expectedPlatIDSQL).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(platID))

	expectedCredentialsSQL := `^INSERT INTO credentials (.+) VALUES \(\d+, \d+, (.+), (.+)\)$`
	Mock.ExpectExec(expectedCredentialsSQL).WillReturnResult(sqlmock.NewResult(1, 1))

	expectedUserbaseSQL := fmt.Sprintf(
		`^INSERT INTO userbase (.+) VALUES \(%d, %d\)$`, // Need to escape the parenthesis or else Regex will think it's a capture group
		userID, clientID,
	)
	Mock.ExpectExec(expectedUserbaseSQL).WillReturnResult(sqlmock.NewResult(1, 1))
	Mock.ExpectCommit()

	// Call the func that we are testing
	actualUserID, err := InsertUserCredentials(DB, CredentialParams{
		ClientID:         clientID,
		PlatformName:     platName,
		UPID:             UPID,
		ConnectionString: connStr,
	})

	// Assertions
	if err != nil {
		t.Errorf("error was not expected when inserting user credentials: %s", err)
	}

	// We make sure that all expectations were met
	if err := Mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t, userID, actualUserID)
}

func TestGetUserConnection_ShouldGetConnection(t *testing.T) {
	userID := 1
	platID := 1
	platName := "fitbit"
	connStr := "oauth2;AC3$$T0K3N;R3FR3$HT0K3N"

	platformIDQuery := fmt.Sprintf("^SELECT id FROM platform WHERE name = '%s'$", platName)
	Mock.ExpectQuery(platformIDQuery).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(platID))

	connStrQuery := fmt.Sprintf(
		"^SELECT connection_string FROM credentials WHERE user_id = %d AND platform_id = %d$",
		userID, platID,
	)
	Mock.ExpectQuery(connStrQuery).WillReturnRows(sqlmock.NewRows([]string{"connection_string"}).AddRow(connStr))

	// Call the func that we are testing
	actualUserConnection, err := GetUserConnection(DB, userID, platName)

	// Assertions
	if err != nil {
		t.Errorf("error was not expected when inserting user credentials: %s", err)
	}

	// We make sure that all expectations were met
	if err := Mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	// Prepare result object
	expectedResult := Connection{
		ConnectionType: "oauth2",
		Parameters: map[string]string{
			"access_token":  "AC3$$T0K3N",
			"refresh_token": "R3FR3$HT0K3N",
		},
	}
	assert.Equal(t, expectedResult, actualUserConnection)
}

func TestGetPlatformDomains_ShouldGetDomains(t *testing.T) {
	cols := []string{
		"name",
		"domain",
	}

	rows := sqlmock.NewRows(cols).
		AddRow("fitbit", "api.fitbit.com").
		AddRow("garmin", "api.garmin.org").
		AddRow("google-fit", "api.google.com").
		AddRow("map-my-tracks", "api.mpt.ca")

	Mock.ExpectQuery("^SELECT name, domain FROM platform$").WillReturnRows(rows)

	actualDomains, err := GetPlatformDomains(DB)
	if err != nil {
		t.Errorf("error was not expected when getting domains: %s", err)
	}

	// We make sure that all expectations were met
	if err := Mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	expectedResult := map[string]string{
		"fitbit":        "api.fitbit.com",
		"garmin":        "api.garmin.org",
		"google-fit":    "api.google.com",
		"map-my-tracks": "api.mpt.ca",
	}

	assert.Equal(t, expectedResult, actualDomains)
}

func TestSignUp_ShouldInsertNewClient(t *testing.T) {
	clientName := "New_Client"
	clientPassword := "Client_Password"

	// Mock SQL rows
	cols := []string{
		"id",
	}
	rows := sqlmock.NewRows(cols).AddRow(1)

	Mock.ExpectQuery(`INSERT INTO client (.+) VALUES (.+)$`).WillReturnRows(rows)

	// call the function we are testing
	clientID, err := CreateNewClient(DB, clientName, clientPassword)
	if err != nil {
		t.Errorf("error was not expected when inserting a client: %s", err)
	}

	// We make sure that all expectations were met
	if err := Mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t, clientID, 1)
}

func TestSignIn_ShouldSignInExistingClient(t *testing.T) {
	clientName := "Registered_Client"
	clientPassword := "Client_Password"
	clientID := 1

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(clientPassword), bcrypt.DefaultCost)

	// Mock SQL rows
	cols := []string{
		"password",
		"id",
	}
	rows := sqlmock.NewRows(cols).AddRow(hashedPassword, clientID)
	Mock.ExpectQuery(fmt.Sprintf("^SELECT password, id FROM client WHERE name = '%s'$", clientName)).WillReturnRows(rows)

	// call the function we are testing
	returnedId, err := SignInClient(DB, clientName, clientPassword)
	if err != nil {
		t.Errorf("error was not expected when signing in a client: %s", err)
	}

	// We make sure that all expectations were met
	if err := Mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t, clientID, returnedId)
}

func TestCheckClientName_ShouldReturnUserId(t *testing.T) {
	clientName := "Searched_Client"
	clientID := 1

	// Mock SQL rows
	cols := []string{
		"id",
	}
	rows := sqlmock.NewRows(cols).AddRow(clientID)

	Mock.ExpectQuery(fmt.Sprintf("^SELECT id FROM client WHERE name = '%s'$", clientName)).WillReturnRows(rows)

	// call the function we are testing
	returnedId, err := CheckClientName(DB, clientName)
	if err != nil {
		t.Errorf("error was not expected when searching for a client: %s", err)
	}

	// We make sure that all expectations were met
	if err := Mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t, clientID, returnedId)
}

func TestAddUserToUserbase_ShouldNotReturnError(t *testing.T) {
	clientID := 1
	userID := 1

	Mock.ExpectExec(`^INSERT INTO userbase (.+) VALUES \(\d+, \d+\)$`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// call the function we are testing
	err := AddUserToUserbase(DB, userID, clientID)
	if err != nil {
		t.Errorf("error was not expected when adding a userID to a client userbase: %s", err)
	}

	// We make sure that all expectations were met
	if err := Mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	assert.Equal(t, err, nil)
}

func TestGetUserInUserbase_ShouldReturnUserID(t *testing.T) {
	clientID := 1
	userID := 1

	// Mock SQL rows
	cols := []string{
		"user_id",
	}
	rows := sqlmock.NewRows(cols).AddRow(userID)

	expectedSQL := fmt.Sprintf("^SELECT user_id FROM userbase WHERE user_id = %d AND client_id = %d$", userID, clientID)
	Mock.ExpectQuery(expectedSQL).WillReturnRows(rows)

	// call the function we are testing
	userIDActual, err := GetUserInUserbase(DB, userID, clientID)
	if err != nil {
		t.Errorf("error was not expected when adding a userID to a client userbase: %s", err)
	}
	assert.Equal(t, userID, userIDActual)
}

func TestUpdateClientCallback_ShouldReturnSuccess(t *testing.T) {
	clientId := 1
	clientCallback := "BrandNewCallback"
	// Mock SQL rows
	cols := []string{
		"id",
	}

	rows := sqlmock.NewRows(cols).AddRow(clientId)
	checkQuery := fmt.Sprintf("SELECT id FROM client WHERE id = %d", clientId)

	// Expect the query to search for the clientID
	Mock.ExpectQuery(checkQuery).WillReturnRows(rows)

	// Expect the query to update the client callback
	Mock.ExpectExec(fmt.Sprintf("UPDATE client SET callback = '%s' WHERE id = %d", clientCallback, clientId)).WillReturnResult(sqlmock.NewResult(1, 1))

	// Call the method we are testing
	result, err := UpdateCallback(DB, clientId, clientCallback)
	// Assertions
	if err != nil {
		t.Errorf("error was not expected when updating client callback: %s", err)
	}
	// We make sure that all expectations were met
	if err := Mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	// Assert that that update passed. (result is true)
	assert.Equal(t, true, result)
}

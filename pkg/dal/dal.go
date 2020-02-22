package dal

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"errors"

	_ "github.com/lib/pq"
)

type Connection struct {
	ConnectionType string
	Parameters     map[string]string
}

type CredentialParams struct {
	ClientID         int
	PlatformName     string
	UPID             string
	ConnectionString string
}

func InitializeDBConn(host string, port int, user, password, dbName string) (*sql.DB, error) {
	connectionString := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbName,
	)

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, err
	}

	// Test connection to database
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func InsertSecretInExistingClient(db *sql.DB, clientID int, secret []byte) (int64, error) {
	// TODO: Use ExecContext instead
	result, err := db.Exec(
		`UPDATE marathon.public.client
				SET secret = $1
				WHERE id = $2`,
		secret,
		clientID,
	)

	if err != nil {
		return 0, err
	}

	rows, _ := result.RowsAffected()
	return rows, nil
}

func GetClientSecret(db *sql.DB, fromClientID int) ([]byte, error) {
	// TODO: Use QueryRowContext instead
	var secret []byte
	err := db.QueryRow("SELECT secret FROM client WHERE id = " + strconv.Itoa(fromClientID)).Scan(&secret)
	if err != nil {
		return nil, err
	}

	return secret, nil
}

func GetUserByPlatformID(db *sql.DB, platformID string, platformName string) (int, error) {
	var userID int

	// check if this user exists in the credentials
	queryString := fmt.Sprintf(
		"SELECT user_id FROM credentials c "+
			"JOIN platform p ON c.platform_id = p.id "+
			"WHERE p.name = %q AND c.upid = %q",
		platformName,
		platformID,
	)

	err := db.QueryRow(queryString).Scan(&userID)

	if err != nil {
		if err == sql.ErrNoRows {
			// there were no rows, but otherwise no error occurred.
			// Return a zero
			return 0, nil
		} else {
			return 0, err
		}
	}

	return userID, nil
}

func InsertUserCredentials(db *sql.DB, params CredentialParams) (int, error) {
	// create a new transaction from the database Connection
	tx, err := db.Begin()

	if err != nil {
		return 0, err
	}

	// we need to either commit or rollback the transaction after it is done.
	defer func() {
		if err != nil {
			// something went wrong, rollback the transaction
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	// the first thing we need to do is to create a new user in the user table
	var userID int
	err = tx.QueryRow(`INSERT INTO marathon.public."user" DEFAULT VALUES RETURNING id`).Scan(&userID)

	if err != nil {
		return 0, err
	}

	// Get platform ID by name
	var platformID int
	platIDQuery := fmt.Sprintf("SELECT id FROM platform WHERE name = %q", params.PlatformName)
	err = db.QueryRow(platIDQuery).Scan(&platformID)
	if err != nil {
		return 0, err
	}

	// add the user into the credentials table
	credentialsQuery := fmt.Sprintf(
		"INSERT INTO credentials "+
			"(user_id, platform_id, upid, connection_string) "+
			"VALUES (%d, %d, %q, %q)",
		userID,
		platformID,
		params.UPID,
		params.ConnectionString,
	)
	_, err = tx.Exec(credentialsQuery)
	if err != nil {
		return 0, err
	}

	// the final step is to add the user to the appropriate row in the userbase table
	userbaseQuery := fmt.Sprintf(
		"INSERT INTO userbase (user_id, client_id) VALUES (%d, %d)", userID, params.ClientID,
	)
	_, err = tx.Exec(userbaseQuery)

	if err != nil {
		return 0, err
	}

	return userID, err // err will be update by the deferred func
}

// TODO: Make it so auth type is not hardcoded in the SQL stmt
func GetUserTokens(db *sql.DB, fromUserID int, platform string) (string, string, error) {
	// get the credentials from the database
	connectionParams, err := GetUserConnection(db, fromUserID, platform)

	if err != nil {
		return "", "", err
	}

	// since we know we are going for tokens, parse them out of the connection struct
	if connectionParams.ConnectionType != "oauth2" {
		return "", "", errors.New("expected Oauth2 authentication type, was instead " + connectionParams.ConnectionType)
	}

	return connectionParams.Parameters["access_token"], connectionParams.Parameters["refresh_token"], nil
}

func GetUserConnection(db *sql.DB, userID int, platformName string) (Connection, error) {
	// Get ID of platform using platform's name
	platIDQuery := fmt.Sprintf("SELECT id FROM platform WHERE name = %q", platformName)
	var platformID int
	err := db.QueryRow(platIDQuery).Scan(&platformID)
	if err != nil {
		return Connection{}, err
	}

	// Get connection string using user's ID and the platform's ID
	connStrQuery := fmt.Sprintf(
		"SELECT connection_string FROM credentials WHERE user_id = %d AND platform_id = %d",
		userID,
		platformID,
	)
	var credentials string
	err = db.QueryRow(connStrQuery).Scan(&credentials)
	if err != nil {
		return Connection{}, err
	}

	// Format the user connection values
	userConnection, err := parseConnectionString(credentials)

	if err != nil {
		return Connection{}, err
	}

	return userConnection, nil
}

func GetPlatformNames(db *sql.DB, fromUserID int) ([]string, error) {
	stmt := fmt.Sprintf(
		"SELECT name FROM platform p "+
			"JOIN credentials c ON p.id = c.platform_id "+
			"WHERE user_id = %d",
		fromUserID,
	)

	// TODO: Use QueryRowContext instead
	rows, err := db.Query(stmt)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var currentPlatform string
	var platforms []string

	for rows.Next() {
		err := rows.Scan(&currentPlatform)
		if err != nil {

			return platforms, err
		}

		platforms = append(platforms, currentPlatform)
	}

	return platforms, nil
}

func parseConnectionString(connectionString string) (Connection, error) {
	returnedConnection := Connection{}

	// split the Connection string into it's components
	connectionParams := strings.Split(connectionString, ";")

	// check what type of Connection string this is
	switch connectionParams[0] {
	case "oauth2":
		// this is an oauth2 Connection, so we will need an access and refresh token
		params := make(map[string]string)
		params["access_token"] = connectionParams[1]
		params["refresh_token"] = connectionParams[2]

		returnedConnection.ConnectionType = "oauth2"
		returnedConnection.Parameters = params
		return returnedConnection, nil
	default:
		// this is not a supported Connection type, bad data in the database
		return returnedConnection, errors.New("Connection type '" + connectionParams[0] + "' unsupported")
	}
}

package dal

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"errors"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
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

func InitializeDBConn(connectionString string) (*sql.DB, error) {
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
		`UPDATE client
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
			"WHERE p.name = '%s' AND c.upid = '%s'",
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

func AddUserToClientUserBase(db *sql.DB, userID int, clientID int) error {
	_, err := db.Exec(`INSERT into userbase(user_id, client_id) VALUES($1,$2)`, userID, clientID)

	if err != nil {
		return err
	}

	return nil

}

func GetUserInUserBase(db *sql.DB, userID int, clientID int) (int, error) {

	// check if this user exists already in the userbase
	queryString := fmt.Sprintf(
		"SELECT user_id FROM userBase "+
			"WHERE user_id = '%d' AND client_id = '%d'",
		userID,
		clientID,
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
	err = tx.QueryRow(`INSERT INTO "user" DEFAULT VALUES RETURNING id`).Scan(&userID)

	if err != nil {
		return 0, err
	}

	// Get platform ID by name
	var platformID int
	platIDQuery := fmt.Sprintf("SELECT id FROM platform WHERE name = '%s'", params.PlatformName)
	err = db.QueryRow(platIDQuery).Scan(&platformID)
	if err != nil {
		return 0, err
	}

	// add the user into the credentials table
	credentialsQuery := fmt.Sprintf(
		"INSERT INTO credentials "+
			"(user_id, platform_id, upid, connection_string) "+
			"VALUES (%d, %d, '%s', '%s')",
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
	platIDQuery := fmt.Sprintf("SELECT id FROM platform WHERE name = '%s'", platformName)
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

func GetPlatformDomains(db *sql.DB) (map[string]string, error) {
	domains := make(map[string]string)

	rows, err := db.Query("SELECT name, domain FROM platform")
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		var name string
		var domain string
		err := rows.Scan(&name, &domain)
		if err != nil {
			return nil, err
		}

		domains[name] = domain
	}

	return domains, nil
}

// CheckClientName takes in a client name, and returns the userId of the client,
// or 0 if no client is using that name
func CheckClientName(db *sql.DB, name string) (int, error) {
	searchQuery := fmt.Sprintf("SELECT id FROM client WHERE name='%s'", name)

	var userId int
	// TODO: Use QueryRowContext instead
	err := db.QueryRow(searchQuery).Scan(&userId)
	if err != nil {
		if err == sql.ErrNoRows {
			// there were no rows, but otherwise no error occurred.
			// that means this name isn't being used in the database
			return 0, nil
		} else {
			return 0, err
		}
	}

	return userId, nil

}

func CreateNewClient(db *sql.DB, name string, password string) error {

	// before we insert the password in the database, we must hash it
	// bcrypt salts this for us, so we don't have to worry about it
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		return err
	}

	insertQuery := fmt.Sprintf("INSERT INTO client (name, password) VALUES('%s','%s')", name, hash)

	// TODO: Use ExecContext instead
	_, err = db.Exec(insertQuery)

	if err != nil {
		return err
	}

	return nil

}

func SignInClient(db *sql.DB, name string, enteredPassword string) (int, error) {

	searchQuery := fmt.Sprintf("SELECT password,id FROM client WHERE name='%s'", name)

	var passwordResult string
	var userId int
	err := db.QueryRow(searchQuery).Scan(&passwordResult, &userId)
	if err != nil {
		return 0, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordResult), []byte(enteredPassword))
	if err != nil {
		return userId, err
	}

	return userId, nil

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

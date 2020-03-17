package dal

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/msgurgel/marathon/pkg/helpers"

	"golang.org/x/oauth2"

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
	result, err := db.Exec( // TODO: Use ExecContext instead
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

	// Check if this user exists in the credentials
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
			// There were no rows, but otherwise no error occurred.
			// Return a zero
			return 0, nil
		} else {
			return 0, err
		}
	}

	return userID, nil
}

func AddUserToUserbase(db *sql.DB, userID int, clientID int) error {
	queryString := fmt.Sprintf("INSERT INTO userbase (user_id, client_id) VALUES (%d, %d)", userID, clientID)

	_, err := db.Exec(queryString)
	if err != nil {
		return err
	}

	return nil
}

func GetUserInUserbase(db *sql.DB, userID int, clientID int) (int, error) {
	// Check if this user exists already in the userbase
	queryString := fmt.Sprintf(
		"SELECT user_id FROM userbase WHERE user_id = %d AND client_id = %d",
		userID,
		clientID,
	)

	err := db.QueryRow(queryString).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			// There were no rows, but otherwise no error occurred.
			// Return a zero
			return 0, nil
		} else {
			return 0, err
		}
	}

	return userID, nil
}

func InsertUserCredentials(db *sql.DB, params CredentialParams) (int, error) {
	// Create a new transaction from the database Connection
	tx, err := db.Begin()

	if err != nil {
		return 0, err
	}

	// We need to either commit or rollback the transaction after it is done.
	defer func() {
		if err != nil {
			// Something went wrong, rollback the transaction
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	// The first thing we need to do is to create a new user in the user table
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

	// Add the user into the credentials table
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

	// The final step is to add the user to the appropriate row in the userbase table
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
func GetUserTokens(db *sql.DB, fromUserID int, platform string) (*oauth2.Token, error) {
	// Get the credentials from the database
	connectionParams, err := GetUserConnection(db, fromUserID, platform)
	if err != nil {
		return &oauth2.Token{}, err
	}

	// Since we know we are going for tokens, parse them out of the connection struct
	if connectionParams.ConnectionType != "oauth2" {
		return &oauth2.Token{}, errors.New("expected Oauth2 authentication type, was instead " + connectionParams.ConnectionType)
	}

	expiry, err := time.Parse(helpers.ISO8601Layout, connectionParams.Parameters["expiry"])
	if err != nil {
		return &oauth2.Token{}, errors.New(
			"failed to convert expiry to time.time – value was " + connectionParams.Parameters["expiry"],
		)
	}

	return &oauth2.Token{
		AccessToken:  connectionParams.Parameters["access_token"],
		RefreshToken: connectionParams.Parameters["refresh_token"],
		TokenType:    connectionParams.Parameters["token_type"],
		Expiry:       expiry,
	}, nil
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

	rows, err := db.Query(stmt) // TODO: Use QueryRowContext instead
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
	searchQuery := fmt.Sprintf("SELECT id FROM client WHERE name = '%s'", name)
	var userId int
	err := db.QueryRow(searchQuery).Scan(&userId) // TODO: Use QueryRowContext instead
	if err != nil {
		if err == sql.ErrNoRows {
			// There were no rows, but otherwise no error occurred.
			// That means this name isn't being used in the database
			return 0, nil
		} else {
			return 0, err
		}
	}

	return userId, nil
}

func CreateNewClient(db *sql.DB, name string, password string) (int, error) {
	// Before we insert the password in the database, we must hash it
	// bcrypt salts this for us, so we don't have to worry about it
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	var clientID int
	insertQuery := fmt.Sprintf("INSERT INTO client (name, password) VALUES ('%s','%s') RETURNING id", name, hash)
	// TODO: Use ExecContext instead
	err = db.QueryRow(insertQuery).Scan(&clientID)
	if err != nil {
		return 0, err
	}

	return clientID, nil
}

func SignInClient(db *sql.DB, name string, enteredPassword string) (int, error) {
	searchQuery := fmt.Sprintf("SELECT password, id FROM client WHERE name = '%s'", name)
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

func UpdateCallback(db *sql.DB, clientID int, newCallback string) (bool, error) {
	clientIDCheck, err := checkClientExistence(db, clientID)
	if err != nil {
		return false, err
	}

	if clientIDCheck {
		// Update the client callback
		updateString := fmt.Sprintf("UPDATE client SET callback = '%s' WHERE id = %d", newCallback, clientID)
		_, err := db.Exec(updateString)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}

func UpdateCredentials(db *sql.DB, userID int, credentialsString string) error {
	updateString := fmt.Sprintf("UPDATE credentials SET connection_string = '%s' WHERE user_id = %d", credentialsString, userID)
	_, err := db.Exec(updateString)
	if err != nil {
		return err
	}

	return nil
}

func UpdateCredentialsUsingOAuth2Tokens(db *sql.DB, userID int, tokens *oauth2.Token) error {
	connStr, err := helpers.FormatConnectionString([]string{
		"oauth2",
		tokens.TokenType,
		tokens.Expiry.Format(helpers.ISO8601Layout),
		tokens.AccessToken,
		tokens.RefreshToken,
	})
	if err != nil {
		return err
	}

	err = UpdateCredentials(db, userID, connStr)
	if err != nil {
		return err
	}

	return nil
}

func checkClientExistence(db *sql.DB, clientID int) (bool, error) {
	clientQuery := fmt.Sprintf("SELECT id FROM client WHERE  id = %d", clientID)

	var clientIDresult int
	err := db.QueryRow(clientQuery).Scan(&clientIDresult)
	if err != nil {
		if err == sql.ErrNoRows {
			// There were no rows, but otherwise no error occurred
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}

func parseConnectionString(connectionString string) (Connection, error) {
	returnedConnection := Connection{}

	// Split the Connection string into it's components
	connectionParams := strings.Split(connectionString, ";")

	// Check what type of Connection string this is
	switch connectionParams[0] {
	case "oauth2":
		// This is an OAuth2 Connection, so we will need an access and refresh token
		params := make(map[string]string)
		params["token_type"] = connectionParams[1]
		params["expiry"] = connectionParams[2]
		params["access_token"] = connectionParams[3]
		params["refresh_token"] = connectionParams[4]

		returnedConnection.ConnectionType = "oauth2"
		returnedConnection.Parameters = params
		return returnedConnection, nil
	default:
		// This is not a supported Connection type, bad data in the database
		return returnedConnection, errors.New("Connection type '" + connectionParams[0] + "' unsupported")
	}
}

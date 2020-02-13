package service

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/msgurgel/marathon/pkg/environment"
)

func InitializeDBConn(config *environment.MarathonConfig) (*sql.DB, error) {
	connectionString := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		config.Database.Host, config.Database.Port, config.Database.User, config.Database.Password, config.Database.DatabaseName,
	)

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, err
	}

	// Test connection
	err = db.PingContext(context.Background())
	if err != nil {
		return nil, err
	}

	return db, nil
}

func InsertSecretInExistingClient(db *sql.DB, clientId int, secret []byte) (int64, error) {
	// TODO: Use ExecContext instead
	result, err := db.Exec(
		`UPDATE marathon.public.client
				SET secret = $1
				WHERE id = $2`,
		secret,
		clientId,
	)

	if err != nil {
		return 0, err
	}

	rows, _ := result.RowsAffected()
	return rows, nil
}

func GetClientSecret(db *sql.DB, fromClientId int) ([]byte, error) {
	// TODO: Use QueryRowContext instead
	var secret []byte
	err := db.QueryRow("SELECT secret FROM client WHERE id = $1", fromClientId).Scan(&secret)
	if err != nil {
		return nil, err
	}

	return secret, nil
}

func CreateFitbitUser(db *sql.DB, accessToken string, refreshToken string, clientId int) (int, error) {

	// create a new transaction from the database connection
	tx, err := db.Begin()

	if err != nil {
		// something went wrong. Return a 0 userId, and an error
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

	var userId int
	// the first thing we need to do is to create a new user in the user table
	err = tx.QueryRow(`INSERT INTO marathon.public.user DEFAULT VALUES RETURNING id`).Scan(&userId)

	if err != nil {
		return 0, err
	}

	// Now we need to do is create a new OAuth2 item in the database for fitbit
	var oauthId int
	// create query with the access and refresh tokens
	// added  ' ' to the access ad refresh tokens because the periods in them seemed to be throwing postgresql off
	insertParams := fmt.Sprintf("INSERT INTO oauth2 (access_token, refresh_token) VALUES ('%s','%s') RETURNING  id", accessToken, refreshToken)
	err = tx.QueryRow(
		insertParams,
	).Scan(&oauthId)

	if err != nil {
		return 0, err
	}

	// the result returned back should be an id that corresponds with the new ID of the Oauth2 id
	// insert into the fitbit table
	fitbitQuery := fmt.Sprintf("INSERT INTO fitbit (user_id, oauth2_id) VALUES (%d,%d)", userId, oauthId)

	_, err = tx.Exec(
		fitbitQuery,
	)
	if err != nil {
		return 0, err
	}

	// the final step is to add the user to the appropriate row in the userbase table
	userbaseQuery := fmt.Sprintf("INSERT INTO marathon.public.userbase (user_id, client_id) VALUES (%d,%d)", userId, clientId)
	_, err = tx.Exec(
		userbaseQuery,
	)

	if err != nil {
		return 0, err
	}

	return userId, err // err will be update by the deferred func
}

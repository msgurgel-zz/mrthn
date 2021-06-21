#!/bin/bash
set -e # Any subsequent(*) commands which fail will cause the shell script to exit immediately

# ======= BEFORE RUNNING THIS SCRIPT =======
# 1. Make sure to run 'bundle install' on the both ruby app directories (integration/sandwich & integration/sandwich/server)
# 2. Set your Postgres env vars in .env.example file and remove .example from the file name
# 3. Run this script from mrthn's home directory ($GOPATH/src/github.com/msgurgel/mrthn) using the following command:
#   ./integration/integration-test.sh

# Get database environment values
source ./.env

# Create log directory
mkdir -p log

# Run setup database script
psql -a $DB_CONNECTION_STRING -f integration/sql/clear-db.sql > log/db_script.log
psql -a $DB_CONNECTION_STRING -f integration/sql/setup-db.sql > log/db_script.log

# Build and run mrthn
go build "$GOPATH"/src/github.com/msgurgel/mrthn/cmd/mrthn
./mrthn &
MRTHN_PID=$!

sleep 1 # Give the server time to start

# Generate JWT for authentication
curl -s "http://localhost:$PORT/get-token?id=1" -H "Origin: https://mrthn.dev"> token.txt

# Run mock third-party server
rackup integration/sandwich/server/config.ru > log/test-server.log 2>&1 &
SERVER_PID=$!

sleep 1 # Give the server time to start

# Run tests!
# We cannot have the script exiting if any tests return as a failure.
# Therefore, have it so the test script always returns true
ruby integration/sandwich/sandwich_test.rb || true


# Exit cleanly
kill -2 $MRTHN_PID
kill -2 $SERVER_PID

rm ./mrthn
rm ./token.txt

# Clear database
psql -a $DB_CONNECTION_STRING -f integration/sql/clear-db.sql > log/db_script.log

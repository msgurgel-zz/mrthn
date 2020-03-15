#!/bin/bash
set -e # Any subsequent(*) commands which fail will cause the shell script to exit immediately

# ======= BEFORE RUNNING THIS SCRIPT =======
# 1. Make sure to run 'bundle install' on the both ruby app directories (integration/sandwich & integration/sandwich/server)
# 2. Set your Postgres env vars in setup-db-env-var.sh.example file and remove .example from the file name
# 3. Run this script from marathon's home directory ($GOPATH/src/github.com/msgurgel/marathon) using the following command:
#   ./integration/integration-test.sh

# Get database environment values
source ./.env

# Create log directory
mkdir -p log

# Run setup database script
psql -a $DB_CONNECTION_STRING -f integration/sql/clear-db.sql > log/db_script.log
psql -a $DB_CONNECTION_STRING -f integration/sql/setup-db.sql > log/db_script.log

# Build and run Marathon
go build "$GOPATH"/src/github.com/msgurgel/marathon/cmd/marathon
./marathon &
MARATHON_PID=$!

sleep 1 # Give the server time to start

# Generate JWT for authentication
curl -s "http://localhost:8080/get-token?id=1" -H "Origin: https://marathon-18119.firebaseapp.com"> token.txt

# Run mock third-party server
rackup integration/sandwich/server/config.ru > log/server.log 2>&1 &
SERVER_PID=$!

sleep 1 # Give the server time to start

# Run tests!
ruby integration/sandwich/sandwich_test.rb

# Exit cleanly
kill -2 $MARATHON_PID
kill -2 $SERVER_PID

rm ./marathon
rm ./token.txt

# Clear database
psql -a $DB_CONNECTION_STRING -f integration/sql/clear-db.sql > log/db_script.log

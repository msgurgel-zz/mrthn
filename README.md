
# mrthn

mrthn (pronounced marathon) is a service that allows you to aggregate the data from multiple Fitness App APIs into simple endpoints. Currently, it supports Google Fitness, Strava, and Fitbit.

[![MIT License](https://img.shields.io/apm/l/atomic-design-ui.svg?)](https://github.com/tterb/atomic-design-ui/blob/master/LICENSEs)

## Environment Variables

To run this project, you will need to set some environment values. To assist you, duplicate the `.env.example` file, rename it to `.env` and set the values within. 

For integration testing, you only need to set `DB_CONNECTION_STRING`.

#### DB_CONNECTION_STRING

This string should point to an instance of PostgreSQL and to the `public` database. It also needs to set `sslmode` to `disable`
```bash
  postgresql://user:secret@localhost/public?sslmode=disable
  
  or

  host=localhost dbname=public user=user sslmode=disable
```

#### CLIENT_ID & CLIENT_SECRET

To allow your instance of mrthn to authenticate with the supported Fitness platforms, you will need to create developer accounts with each of them and generate OAuth2 credentials.

- [Fitbit](https://dev.fitbit.com/getting-started/)
- [Google Fit](https://developers.google.com/fit/rest/v1/get-started)
- [Strava](https://developers.strava.com/docs/getting-started/)

Explanation for other environment variables coming soon...
## Database Set Up

To run this service or its integration tests, you will need to set up a db instance

- Have a PostgreSQL server set up. **Note that mrthn will clear your `public` database.**
- Run `db/db_create.sql` and `db/user_data.sql` in your PostgreSQL instance. Example: 
```bash
  psql -a postgresql://user:secret@localhost/public -f db/user_data.sql)
```
- You're all set!
## Run Locally

1. Clone the project

```bash
  # run this command within $GOPATH/src/github.com/msgurgel/
  git clone https://github.com/msgurgel/mrthn.git
```

2. Go to the project directory

```bash
  cd mrthn
```

3. Install dependencies

```bash
  go get ./...
```

4. Set environment variables. See [Environment Variables](#environment-variables) section.
5. Set up your database. See [Database Set Up](#database-set-up) section.

5. Compile the code

```bash
  go build ./cmd/mrthn
```

6. Run the executable

```bash
  ./mrthn --log-to-stderr
```    

  
## Running Tests

To run unit tests, run the following command

```bash
  go test ./...
```

To run the integration tests, you will need to do some setup first:

#### Set up your local database

See [Database Set Up](#database-set-up) section.


#### Install `sandwich` depencies

`sandwich` is a small ruby app that runs the integration tests (see [README]()). To install its depencies, run the following command inside of `integration/sandwich` and `integration/sandwich/server`
 
```bash
  bundle install
```

#### Set environment variables

See [Environment Variables](#environment-variables) section.

#### Run integration test script

Finally, run the integration test script from the project's home directory

```bash
  # in $GOPATH/src/github.com/msgurgel/mrthn  
  ./integration/integration-test.sh
```

If anything goes wrong, you can check the `log/` directory for more information.
## API Reference

### Public Endpoints

#### Check if service is up

```http
  GET /
```

#### Get daily calories

```http
  GET /user/${userId}/calories/daily?date=${date}
```

| Path Parameter | Type     | Description                       |
| :------------- | :------- | :-------------------------------- |
| `userId`       | `integer`| **Required**. Id of the user to fetch data from |

| Query Parameter | Type     | Description                       |
| :-------------- | :------- | :-------------------------------- |
| `date`          | `date`   | **Required**. Day to get data from. Format is YYYY-MM-DD |
| `largestOnly`   | `bool`   | Set to `true` to return data from only the platform with the largest value |



#### Get daily steps

```http
  GET /user/${userId}/steps/daily?date=${date}
```

| Path Parameter | Type     | Description                       |
| :------------- | :------- | :-------------------------------- |
| `userId`       | `integer`| **Required**. Id of the user to fetch data from |

| Query Parameter | Type     | Description                       |
| :-------------- | :------- | :-------------------------------- |
| `date`          | `date`   | **Required**. Day to get data from. Format is YYYY-MM-DD |
| `largestOnly`   | `bool`   | Set to `true` to return data from only the platform with the largest value |


#### Get daily distance travelled

```http
  GET /user/${userId}/steps/distance?date=${date}
```

| Path Parameter | Type     | Description                       |
| :------------- | :------- | :-------------------------------- |
| `userId`       | `integer`| **Required**. Id of the user to fetch data from |

| Query Parameter | Type     | Description                       |
| :-------------- | :------- | :-------------------------------- |
| `date`          | `date`   | **Required**. Day to get data from. Format is YYYY-MM-DD |
| `largestOnly`   | `bool`   | Set to `true` to return data from only the platform with the largest value |


#### Get distance travelled over a period of time

```http
  GET /user/${userId}/distance/over-period?period=${period}
```

| Path Parameter | Type     | Description                       |
| :------------- | :------- | :-------------------------------- |
| `userId`       | `integer`| **Required**. Id of the user to fetch data from |

| Query Parameter | Type     | Description                       |
| :-------------- | :------- | :-------------------------------- |
| `period`        | `period`   | **Required**. Period of time to get data from. Possible values: "1d", "7d", "30d", "1w", "1m", "3m", "6m" |



#### Private Endpoints

Will add this section soon! 🔜
  
## Authors

- [@Manuel9550](https://www.github.com/Manuel9550)
- [@msgurgel](https://www.github.com/msgurgel)

  
## Feedback

If you have any feedback or questions, please reach out to me at me@magu.dev

  

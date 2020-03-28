DROP SCHEMA public CASCADE;
CREATE SCHEMA public;
SET SEARCH_PATH = "public";

CREATE TABLE "user"(
    id SERIAL PRIMARY KEY
);
CREATE TABLE client(
    id     SERIAL       PRIMARY KEY,
    name   VARCHAR(50)  NOT NULL,
    secret BYTEA,
    password TEXT NOT NULL,
    callback TEXT
);
CREATE TABLE userbase(
    id        SERIAL  PRIMARY KEY,
    user_id   INTEGER REFERENCES "user"(id),
    client_id INTEGER REFERENCES client(id)
);
CREATE TABLE platform(
    id          SERIAL      PRIMARY KEY,
    name        VARCHAR(64) NOT NULL,
    domain      VARCHAR(128) NOT NULL
);
CREATE INDEX plat_name_index ON platform(name);
CREATE TABLE credentials(
    id                SERIAL      PRIMARY KEY,
    user_id           INTEGER     REFERENCES "user"(id),
    platform_id       INTEGER     REFERENCES platform(id),
    upid              BYTEA NOT NULL, -- User-Platform ID (ID of an user for an specific platform)
    connection_string TEXT        NOT NULL
);
CREATE INDEX credentials_upid_index ON credentials(upid);
-- Insert initial setup values
INSERT INTO client (name, password, callback)
VALUES ('Passive Marathon', 'bad_hash', 'test_callback');

INSERT INTO platform (name, domain) VALUES ('google', 'https://www.googleapis.com/fitness/v1');
INSERT INTO platform (name, domain) VALUES ('fitbit', 'https://api.fitbit.com/1');
INSERT INTO platform (name, domain) VALUES ('strava', 'https://www.strava.com/api/v3');

DROP database marathon;
CREATE DATABASE marathon;
SET SEARCH_PATH = "public";

CREATE TABLE "user"(
    id SERIAL PRIMARY KEY
);
​
CREATE TABLE client(
    id     SERIAL       PRIMARY KEY,
    name   VARCHAR(50)  NOT NULL,
    secret BYTEA
);
​
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
    upid              VARCHAR(32) NOT NULL, -- User-Platform ID (ID of an user for an specific platform)
    connection_string TEXT        NOT NULL
);
CREATE INDEX credentials_upid_index ON credentials(upid);

-- Insert initial setup values
INSERT INTO client (name)
VALUES ('Passive Marathon');

INSERT INTO platform (name, domain) VALUES ('fitbit', 'https://api.fitbit.com/1')

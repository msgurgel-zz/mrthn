DROP database marathon;
CREATE DATABASE marathon;
SET SEARCH_PATH = "public";
​

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
​
CREATE TABLE credentials(
    id                SERIAL      PRIMARY KEY,
    user_id           INTEGER     REFERENCES "user"(id),
    platform_name     VARCHAR(64) NOT NULL,
    platform_id       VARCHAR(32) UNIQUE NOT NULL,
    connection_string TEXT        NOT NULL

);
CREATE INDEX platform_name_index ON credentials(platform_name);

INSERT INTO client (name)
VALUES ('Passive Marathon');
-- Restart DB
DELETE FROM credentials;
DELETE FROM platform;
DELETE FROM userbase;
DELETE FROM client;
DELETE FROM user_data;
DELETE FROM "user";

ALTER SEQUENCE credentials_id_seq RESTART WITH 1;
ALTER SEQUENCE platform_id_seq RESTART WITH 1;
ALTER SEQUENCE userbase_id_seq RESTART WITH 1;
ALTER SEQUENCE client_id_seq RESTART WITH 1;
ALTER SEQUENCE user_id_seq RESTART WITH 1;


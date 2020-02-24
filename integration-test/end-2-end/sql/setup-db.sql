-- Insert initial setup values
INSERT INTO "user" DEFAULT VALUES; -- Creates User 1
INSERT INTO platform (name, domain) VALUES ('fitbit', 'http://localhost:9292/fitbit'); -- Creates mock Fitbit
INSERT INTO credentials (user_id, platform_id, upid, connection_string) VALUES (1, 1, 'A1B2C3', 'oauth2;ACC3$$T0K3N;R3FR3$HT0K3N');
INSERT INTO client (name) VALUES ('Sandwich'); -- Creates our test app client
INSERT INTO userbase (user_id, client_id) VALUES (1, 1);


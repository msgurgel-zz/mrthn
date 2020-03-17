-- Insert initial setup values
INSERT INTO "user" DEFAULT VALUES; -- Creates User 1
INSERT INTO platform (name, domain) VALUES ('fitbit', 'http://localhost:9292/fitbit'); -- Creates mock Fitbit
INSERT INTO credentials (user_id, platform_id, upid, connection_string) VALUES (1, 1, 'A1B2C3', 'oauth2;Bearer;3005-04-23T04:20:00-0400;ACC3$$T0K3N;R3FR3$HT0K3N'); -- This token cannot expire!
INSERT INTO client (name, password, callback) VALUES ('Sandwich', 'Sandwich_Password', 'Test_Callback'); -- Creates our test app client
INSERT INTO userbase (user_id, client_id) VALUES (1, 1);


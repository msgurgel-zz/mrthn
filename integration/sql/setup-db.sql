-- Insert initial setup values
INSERT INTO "user" DEFAULT VALUES; -- Creates User 1
INSERT INTO "user" DEFAULT VALUES;
INSERT INTO "user" DEFAULT VALUES;
INSERT INTO "user" DEFAULT VALUES;
INSERT INTO platform (name, domain) VALUES ('fitbit', 'http://localhost:9292/fitbit'); -- Creates mock Fitbit
INSERT INTO platform (name, domain) VALUES ('google', 'http://localhost:9292/google/fitness/v1/'); -- Creates mock Google
INSERT INTO platform (name, domain) VALUES ('strava', 'http://localhost:9292/strava/'); -- Creates mock Strava
INSERT INTO credentials (user_id, platform_id, upid, connection_string) VALUES (1, 1, 'A1B2C3', 'oauth2;Bearer;3005-04-23T04:20:00-0400;ACC3$$T0K3N;R3FR3$HT0K3N'); -- All credential tokens cannot expire!
INSERT INTO credentials (user_id, platform_id, upid, connection_string) VALUES (2, 2, 'testAccount@gmail.com', 'oauth2;Bearer;3005-04-23T04:20:00-0400;ACC3$$T0K3NGOOGLE;R3FR3$HT0K3NGOOGLE');
INSERT INTO credentials (user_id, platform_id, upid, connection_string) VALUES (3, 1, 'F5H7J9', 'oauth2;Bearer;3005-04-23T04:20:00-0400;ACC3$$T0K3NFITBIT2;R3FR3$HT0K3FITBIT2');
INSERT INTO credentials (user_id, platform_id, upid, connection_string) VALUES (3, 2, 'MULTIPLE_PLATFORMS@gmail.com', 'oauth2;Bearer;3005-04-23T04:20:00-0400;ACC3$$T0K3NGOOGLE2;R3FR3$HT0K3NGOOGLE2');
INSERT INTO credentials (user_id, platform_id, upid, connection_string) VALUES (3, 3, 'G5J84', 'oauth2;Bearer;3005-04-23T04:20:00-0400;ACC3$$T0K3NSTRAVA;R3FR3$HT0K3NSTRAVA');
INSERT INTO credentials (user_id, platform_id, upid, connection_string) VALUES (4, 3, 'G5J84', 'oauth2;Bearer;3005-04-23T04:20:00-0400;ACC3$$T0K3NSTRAVA2;R3FR3$HT0K3NSTRAVA2');
INSERT INTO client (name, password, callback) VALUES ('Sandwich', 'Sandwich_Password', 'Test_Callback'); -- Creates our test app client
INSERT INTO userbase (user_id, client_id) VALUES (1, 1);
INSERT INTO userbase (user_id, client_id) VALUES (2, 1);
INSERT INTO userbase (user_id, client_id) VALUES (3, 1);
INSERT INTO userbase (user_id, client_id) VALUES (4, 1);

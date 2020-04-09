SET SEARCH_PATH = "public";

CREATE OR REPLACE FUNCTION random_between(low INT, high INT)
   RETURNS INT AS
$$
BEGIN
   RETURN floor(random()* (high-low + 1) + low);
END;
$$ language 'plpgsql' STRICT;

CREATE OR REPLACE PROCEDURE InsertUserData(userID BIGINT, platformID INTEGER, insertDate DATE)
LANGUAGE 'plpgsql'
AS $$
DECLARE
    -- Max and Min values for calories, steps and distance
    MAX_CALORIES INTEGER := 3000;
    MIN_CALORIES INTEGER := 1500;

    MAX_STEPS INTEGER := 15000;
    MIN_STEPS INTEGER := 1000;

    MAX_DISTANCE FLOAT := 0.5;
    MIN_DISTANCE FLOAT := 5.5;

    user_calories INTEGER := random_between(MIN_CALORIES,MAX_CALORIES);
    user_steps INTEGER := random_between(MIN_STEPS,MAX_STEPS);
    user_distance FLOAT := random()* (MAX_DISTANCE-MIN_DISTANCE + 1) + MIN_DISTANCE;
BEGIN
    INSERT INTO user_data (user_id, platform_id,date, steps, calories, distance)
    VALUES (userID,platformID,insertDate,user_steps, user_calories,user_distance);
END
$$;

CREATE OR REPLACE PROCEDURE GenerateUserData(userID BIGINT, insertDate DATE)
LANGUAGE 'plpgsql'
AS $$
DECLARE

BEGIN
    CALL InsertUserData(userID, 1, insertDate);
    CALL InsertUserData(userID, 2, insertDate);
    CALL InsertUserData(userID, 3, insertDate);
END
$$;

-- The actual stored procedure that will generate a month's worth of Data
CREATE OR REPLACE PROCEDURE GenerateData()
LANGUAGE 'plpgsql'
AS $$
DECLARE
   counter INTEGER := 0 ;
   numDays INTEGER := 31;
   startDate DATE := CURRENT_DATE - 30;
   numUsers INTEGER := 14;
   currentUsers INTEGER := 0;
   currentUserID BIGINT := 0;
BEGIN

    LOOP
    -- Loop once for every user we created
    EXIT WHEN currentUsers > numUsers;
        counter := 0;
        startDate := CURRENT_DATE - 30;

        INSERT INTO "user" DEFAULT VALUES;
        currentUserID := CURRVAL('user_id_seq');
        LOOP
            -- Loop once for every day we want to generate data for
            EXIT WHEN counter > numDays;

            -- GenerateUserData creates three entries for the specified date, one for each platform
            CALL GenerateUserData(currentUserID, startDate);

            counter := counter + 1;
            startDate := startDate + 1;
        END LOOP;

        currentUsers := currentUsers + 1;
    END LOOP;
END
$$;

-- Call the stored procedure to generate random data for the last 30 days
CALL GenerateData();


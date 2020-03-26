require "minitest/autorun"
require "minitest/reporters"
Minitest::Reporters.use! Minitest::Reporters::ProgressReporter.new

require 'net/http'
require 'json'
require 'httparty'

class SandwichTest < Minitest::Test
    def setup
        token_file = File.open('token.txt')
        @jwt = token_file.read
        token_file.close
    end

    def test_get_steps_fitbit
        response = HTTParty.get('http://localhost:8080/user/1/steps?date=2020-02-13', {
            headers: {
                "User-Agent" => "Sandwich",
                "Authorization" => "Bearer #{@jwt}",
            }
        })

        parsed = JSON.parse(response.body)
        assert_equal 1, parsed["id"]
        assert_equal 'fitbit', parsed["steps"][0]["platform"]
        assert_equal 2020, parsed["steps"][0]["value"]
    end

    def test_get_steps_google
        response = HTTParty.get('http://localhost:8080/user/2/steps?date=2020-02-13', {
            headers: {
                "User-Agent" => "Sandwich",
                 "Authorization" => "Bearer #{@jwt}",
                }
            })

            parsed = JSON.parse(response.body)
            assert_equal 2, parsed["id"]
            assert_equal 'google', parsed["steps"][0]["platform"]
            assert_equal 500, parsed["steps"][0]["value"]
    end

    def test_get_calories_google
        response = HTTParty.get('http://localhost:8080/user/2/calories?date=2020-02-13', {
            headers: {
                "User-Agent" => "Sandwich",
                 "Authorization" => "Bearer #{@jwt}",
                   }
            })

            parsed = JSON.parse(response.body)
            assert_equal 2, parsed["id"]
            assert_equal 'google', parsed["calories"][0]["platform"]
            assert_equal 1635, parsed["calories"][0]["value"]
    end

    def test_get_distance_google
        response = HTTParty.get('http://localhost:8080/user/2/distance?date=2020-02-13', {
            headers: {
                "User-Agent" => "Sandwich",
                "Authorization" => "Bearer #{@jwt}",
                }
            })

            parsed = JSON.parse(response.body)
            assert_equal 2, parsed["id"]
            assert_equal 'google', parsed["distance"][0]["platform"]
            assert_equal 3.456, parsed["distance"][0]["value"]
    end

    def test_get_distance_strava
            response = HTTParty.get('http://localhost:8080/user/4/distance?date=2020-02-13', {
                headers: {
                    "User-Agent" => "Sandwich",
                    "Authorization" => "Bearer #{@jwt}",
                    }
                })

                parsed = JSON.parse(response.body)
                assert_equal 4, parsed["id"]
                assert_equal 'strava', parsed["distance"][0]["platform"]
                assert_equal 1.304, parsed["distance"][0]["value"]
    end

    def test_get_calories_all_platforms
            response = HTTParty.get('http://localhost:8080/user/3/calories?date=2020-02-13', {
                headers: {
                    "User-Agent" => "Sandwich",
                    "Authorization" => "Bearer #{@jwt}",
                    }
                })

                parsed = JSON.parse(response.body)
                assert_equal 3, parsed["id"]

                #it appears fitbit will come before google in the return object
                assert_equal 'fitbit', parsed["calories"][0]["platform"]
                assert_equal 1010, parsed["calories"][0]["value"]

                assert_equal 'google', parsed["calories"][1]["platform"]
                assert_equal 1635, parsed["calories"][1]["value"]

                assert_equal 'strava', parsed["calories"][2]["platform"]
                assert_equal 938, parsed["calories"][2]["value"]
    end

    def test_get_max_calories
                response = HTTParty.get('http://localhost:8080/user/3/calories/largest?date=2020-02-13', {
                    headers: {
                        "User-Agent" => "Sandwich",
                        "Authorization" => "Bearer #{@jwt}",
                        }
                    })

                    parsed = JSON.parse(response.body)
                    assert_equal 3, parsed["id"]

                    # There should only be the google platform returned, because it has more calories than Fitbit
                    assert_equal 'google', parsed["calories"][0]["platform"]
                    assert_equal 1635, parsed["calories"][0]["value"]
    end

    def test_get_max_steps
                    response = HTTParty.get('http://localhost:8080/user/3/steps/largest?date=2020-02-13', {
                        headers: {
                            "User-Agent" => "Sandwich",
                            "Authorization" => "Bearer #{@jwt}",
                            }
                        })

                        parsed = JSON.parse(response.body)
                        assert_equal 3, parsed["id"]

                        # The Fitbit platform has more steps than Google, so it should return the Fitbit Amount
                        assert_equal 'fitbit', parsed["steps"][0]["platform"]
                        assert_equal 2020, parsed["steps"][0]["value"]
    end

    def test_get_max_distance
                        response = HTTParty.get('http://localhost:8080/user/3/distance/largest?date=2020-02-13', {
                            headers: {
                                "User-Agent" => "Sandwich",
                                "Authorization" => "Bearer #{@jwt}",
                                }
                            })

                            parsed = JSON.parse(response.body)
                            assert_equal 3, parsed["id"]

                            # Google has more distance than fitbit
                            assert_equal 'google', parsed["distance"][0]["platform"]
                            assert_equal 3.456, parsed["distance"][0]["value"]
        end

    def test_client_signup_name_already_taken
        response = HTTParty.post(
            'http://localhost:8080/signup',
            multipart: true,
            body: {
                name: 'Sandwich',
                password: 'whatever'
            },
            headers: {
                'Origin' => 'https://marathon-18119.firebaseapp.com'
            }
        )
        parsed = JSON.parse(response.body)

        assert_equal false, parsed["success"]
        assert_equal 'Client name already taken', parsed["error"]
    end

    def test_client_signup_no_name
        response = HTTParty.post(
            'http://localhost:8080/signup',
            multipart: true,
            body: {
                password: 'whatever'
            },
            headers: {
                'Origin' => 'https://marathon-18119.firebaseapp.com'
            }
        )
        parsed = JSON.parse(response.body)

        assert_equal false, parsed["success"]
        assert_equal "Expected parameter 'name' in request", parsed["error"]
    end

    def test_client_signup_no_password
        response = HTTParty.post(
            'http://localhost:8080/signup',
            multipart: true,
            body: {
                name: 'no_password_client',
            },
            headers: {
                'Origin' => 'https://marathon-18119.firebaseapp.com'
            }
        )
        parsed = JSON.parse(response.body)

        assert_equal false, parsed["success"]
        assert_equal "Expected parameter 'password' in request", parsed["error"]
    end

    def test_client_signup_new_client
        response = HTTParty.post(
            'http://localhost:8080/signup',
            multipart: true,
            body: {
                name: 'new_name',
                password: 'pass12345'
            },
            headers: {
                'Origin' => 'https://marathon-18119.firebaseapp.com'
            }
        )

        parsed = JSON.parse(response.body)

        assert_equal true, parsed["success"]
        assert_nil parsed["error"]
    end

    def test_client_signin_wrong_password
        response = HTTParty.post('http://localhost:8080/signin', {
            :body => {
                :name => 'Sandwich',
                :password => 'Bad_Password'
            },
            :headers => {
                'Content-Type' => 'application/x-www-form-urlencoded',
                'Origin' => 'https://marathon-18119.firebaseapp.com'
            }
        })
        parsed = JSON.parse(response.body)

        assert_equal false, parsed["success"]
        assert_equal 'Incorrect password', parsed["error"]
    end

    def test_client_update_callback_incorrect_client
            response = HTTParty.post('http://localhost:8080/client/34/callback', {
                :body => {
                    :callback => 'Sandwich'
                },
                :headers => {
                    'Content-Type' => 'application/x-www-form-urlencoded',
                    'Origin' => 'https://marathon-18119.firebaseapp.com'
                }
            })
            parsed = JSON.parse(response.body)

            assert_equal false, parsed["success"]
            assert_equal 'clientID does not match any registered client', parsed["error"]
    end

    def test_client_update_callback_non_integer_clientID
                response = HTTParty.post('http://localhost:8080/client/notaninteger/callback', {
                    :body => {
                        :callback => 'Sandwich'
                    },
                    :headers => {
                        'Content-Type' => 'application/x-www-form-urlencoded',
                        'Origin' => 'https://marathon-18119.firebaseapp.com'
                    }
                })
                parsed = JSON.parse(response.body)

                assert_equal false, parsed["success"]
                assert_equal 'clientID must be an integer', parsed["error"]
    end

    def test_client_update_callback_correct_call
                    response = HTTParty.post('http://localhost:8080/client/1/callback', {
                        :body => {
                            :callback => 'Sandwich_callback'
                        },
                        :headers => {
                            'Content-Type' => 'application/x-www-form-urlencoded',
                            'Origin' => 'https://marathon-18119.firebaseapp.com'
                        }
                    })
                    parsed = JSON.parse(response.body)

                    assert_equal true, parsed["success"]
                    assert_equal 'Sandwich_callback', parsed["updatedCallback"]
    end

end
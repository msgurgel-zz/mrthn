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
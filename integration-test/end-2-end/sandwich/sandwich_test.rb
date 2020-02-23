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
                "Authorization" => "Bearer #{@jwt}"
            }
        })
        parsed = JSON.parse(response.body)

        assert_equal parsed["id"], 1
        assert_equal parsed["steps"][0]["platform"], 'fitbit'
        assert_equal parsed["steps"][0]["value"], 2020
    end
end
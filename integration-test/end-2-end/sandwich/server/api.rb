require 'grape'

module TestServer
    class API < Grape::API
        format :json

        # Mocks Fitbit endpoints
        resource :fitbit do
            resource :user do
                route_param :user_id do
                    resource :activities do
                        resource :date do
                            get :"2020-02-13.json" do
                                {
                                    summary: {
                                        caloriesOut: 1010,
                                        steps: 2020,
                                        distances: [
                                                    {"activity":"total","distance":2.63},
                                                    {"activity":"tracker","distance":2.63}
                                                    ]
                                    }
                                }
                            end
                        end
                    end
                end
           end
       end

    end
end
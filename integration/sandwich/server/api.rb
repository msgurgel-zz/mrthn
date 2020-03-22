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


       # Mocks Google endpoints
       resource :google do
            resource :fitness do
                resource :v1 do
                    resource :users do
                        resource :me do
                            post :'dataset:aggregate' do

                                 valueToReturn = 0
                                 returnValueType = "intVal"
                                 dataTypename = "caloriesExpended"

                                 case params["aggregateBy"][0]["dataSourceId"]
                                 when "derived:com.google.step_count.delta:com.google.android.gms:estimated_steps"
                                    valueToReturn = 500
                                    returnValueType = "intVal"

                                 when "derived:com.google.calories.expended:com.google.android.gms:merge_calories_expended"
                                    valueToReturn = 1635
                                    returnValueType = "fpVal"

                                 when "derived:com.google.distance.delta:com.google.android.gms:merge_distance_delta"
                                    valueToReturn = 3456
                                    returnValueType = "fpVal"
                                 else
                                    raise 'Improper dataSourceID'
                                 end

                                 {
                                    bucket: [
                                        {
                                            dataset: [
                                                {
                                                    dataSourceId: params["aggregateBy"][0]["dataSourceId"],
                                                    point: [
                                                        {
                                                            value: [
                                                                {
                                                                    mapVal: [],
                                                                    "#{returnValueType}": valueToReturn
                                                                 }
                                                            ],
                                                        }
                                                    ]
                                                }
                                            ]
                                        }
                                    ]
                                 }
                            end
                        end
                    end
                end
            end
       end
    end
end
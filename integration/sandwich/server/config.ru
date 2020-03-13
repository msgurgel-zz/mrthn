# config.ru
require 'rack'
require_relative 'api'

run TestServer::API # Mounts on top of Rack.
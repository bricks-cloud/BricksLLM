## 1.31.1 - 2024-07-02
### Fixed
- Fixed OpenAI transcription endpoint not working error

## 1.31.0 - 2024-07-02
### Added
- Added support for storing metadata per request via the `X-METADATA` header

## 1.30.0 - 2024-
06-27
### Added
- Added deletion endpoint for routes
- Added support for new models from OpenAI and Azure OpenAI to routes

## 1.29.0 - 2024-06-24
### Added
- Added support for Anthropic sonnet 3.5
- Added retry strategy to routes
- Added support for Azure OpenAI completions endpoint

### Changed
- Changed to using sha256 hashing algorithm for computing cache key within routes

## 1.28.4 - 2024-06-12
### Added
- Added filtering by status code for events v2 API

## 1.28.3 - 2024-06-12
### Fixed
- Fixed issues with updating provider settings

## 1.28.2 - 2024-06-12
### Fixed
- Removed unused logs

### Fixed
- Changed from using response model to using request model as the source of truth for customized pricing

## 1.28.1 - 2024-06-10
### Fixed
- Removed debug logs

## 1.28.0 - 2024-06-10
### Added
- Added support for customized model pricing for Azure OpenAI, OpenAI, vLLM and Deep Infra

## 1.27.0 - 2024-06-05
### Added
- Added support for OpenAI batch APIs

### Fixed
- Fixed event not being recorded when request body is empty and request logging is turned on

## 1.26.3 - 2024-06-05
### Added
- Added support for `gpt-4o` inside Azure OpenAI

### Fixed
- Fixed Azure OpenAI token counting when streaming

## 1.26.2 - 2024-05-30
### Fixed
- Fixed configuration files priority issue

## 1.26.1 - 2024-05-30
### Fixed
- Removed unused env variables
- Fixed `.env` file cannot be openned error

## 1.26.0 - 2024-05-25
### Added
- Added `.env` and `json` file configurations for env variables

### Fixed
- Removed unused funcition variables such as `cid`

### Changed
- Updated `go` version to `1.22`

## 1.25.3 - 2024-05-24
### Fixed
- Fixed token counting for `gpt-4o`

## 1.25.2 - 2024-05-23
### Fixed
- Reconfigured logger to always log `correlationId`

## 1.25.1 - 2024-05-23
### Fixed
- Reconfigured logger to always log `correlationId`

## 1.25.0 - 2024-05-23
### Added
- Added `requestFormat` and `requestParams` to routes

## 1.24.0 - 2024-05-13
### Added
- Added policy integration with assistant APIs

## 1.23.2 - 2024-05-10
### Fixed
- Fixed issue with policy running even if policy input is empty
  
## 1.23.1 - 2024-05-10
### Added
- Added support for gpt-4o

## 1.23.0 - 2024-05-09
### Fixed
- Fixed policy bug where regex redaction rules run even when there isn't a match

## 1.22.0 - 2024-05-09
### Added
- Added `redacted` as an option for filters

## 1.21.0 - 2024-05-06
### Changed
- Updated v2 events API to allow more array params such as `userIds` instead of `userId`

## 1.20.0 - 2024-05-01
### Fixed
- Fixed an issue with forwarding requests to OpenAI's Assistant APIs

## 1.19.0 - 2024-05-01
### Added
- Added v2 get events API

## 1.18.3 - 2024-04-22
### Added
- Added a new filter for retrieving the total number of keys for the get keys v2 API

### Changed
- Changed get keys V2 API response format

## 1.18.2 - 2024-04-22
### Fixed
- Fixed PII detection bug

## 1.18.1 - 2024-04-21
### Added
- Added new env variable for setting AWS region `AMAZON_REGION`
- Added new `name` and `order` filters to get keys V2 API
- Added new `name` and `revoked` filters to top keys API

### Fixed
- Fixed top keys API for not returning keys with no spend

## 1.18.0 - 2024-04-17
### Added
- Added a new API `/api/v2/key-management/keys` for retrieving keys
- Added a new API `/api/reporting/top-keys` for retrieving key IDs sorted by spend
- Added a new API `/api/users` for cerating a user
- Added a new API `/api/users/:id` for updating a user via user ID 
- Added a new API `/api/users` for updating a user via tags and user ID
- Added a new API `/api/users` for getting users

## 1.17.0 - 2024-04-11
### Added
- Added new admin API endpoints for retriving `customIds` and `userIds`

## 1.16.0 - 2024-04-10
### Added
- Added Deepinfra integration

## 1.15.4 - 2024-04-09
### Changed
- Changed update provider setting behavior to only do partial updates for `setting` field

## 1.15.3 - 2024-04-09
### Added
- Provider settings APIs start returning `setting` field without containing `apikey`

## 1.15.2 - 2024-04-09
### Fixed
- Fixed an issue with `apikey` being required for making vLLM requests

## 1.15.1 - 2024-04-07
### Added
- Added support for `apikey` in vLLM integration

## 1.15.0 - 2024-04-07
### Added
- Added vLLM integration

## 1.14.3 - 2024-04-03
### Fixed
- Fixed revoking key without hitting cost limit error

## 1.14.2 - 2024-04-02
### Added
- Added fetch from DB if a provider setting associated with a key is not found in memory

## 1.14.1 - 2024-04-01
### Fixed
- Fixed user Id not being captured with Anthropic's new messages endpoint

## 1.14.0 - 2024-03-29
### Added
- Added integration with Anthropic's new messages endpoint

## 1.13.12 - 2024-03-25
### Fixed
- Fixed a db query issue with the new reporting endpoint

## 1.13.11 - 2024-03-25
### Added
- Added new reporting endpoint for retrieving metric data points by day

## 1.13.10 - 2024-03-25
### Added
- Added a new key field called `isKeyNotHashed`

## 1.13.9 - 2024-03-23
### Fixed
- Fixed http response exceeding `1GB` when receiving a streaming error from OpenAI

## 1.13.8 - 2024-03-20
### Changed
- Updated error messages for authentication errors

## 1.13.7 - 2024-03-20
### Changed
- Changed error message for authentication errors

## 1.13.6 - 2024-03-20
### Added
- Added more detailed logs for authentication errors

## 1.13.5 - 2024-03-20
### Added
- Added stats when key is not found in cache

## 1.13.4 - 2024-03-20
### Added
- Added fallback when key is not found in cache

## 1.13.3 - 2024-03-19
### Fixed
- Fixed `ttl` behavior not to disable keys

## 1.13.2 - 2024-03-18
### Changed
- Updated default postgresql read operation timeout from `15s` to `2m`

## 1.13.1 - 2024-03-15
### Fixed
- Fixed `nil` pointer error
s
## 1.13.0 - 2024-03-14
### Added
- Added `policy` APIs

## 1.12.3 - 2024-03-07
### Added
- Started supporting storing streaming responses in bytes as part of `event`

### Fixed
- Fixed issue with event not being recorded if `shouldLogResponse` is set to `true` for streaming responses

## 1.12.2 - 2024-03-05
### Added
- Updated update key API to support setting `costLimitInUsdOverTime`, `costLimitInUsd` and `rateLimitOverTime` to 0

## 1.12.1 - 2024-02-28
### Added
- Added querying keys by `keyIds`
- Increased default postgres DB read timeout to `15s` and write timeout to `5s`

## 1.12.0 - 2024-02-28
### Added
- Added setting rotation feature to key

## 1.11.0 - 2024-02-28
### Added
- Added cost tracking for OpenAI audio endpoints
- Added inference cost tracking for OpenAI finetune models

## 1.10.0 - 2024-02-21
### Added
- Added `userId` as a new filter option for get events API endpoint
- Added option to store request and response using keys

## 1.9.6 - 2024-02-18
### Added
- Added support for updating key cost limit and rate limit

### Changed
- Removed validation to updating revoked key field

## 1.9.5 - 2024-02-18
### Added
- Added new model "gpt-4-turbo-preview" and "gpt-4-vision-preview" to the cost map

## 1.9.4 - 2024-02-16
### Added
- Added support for calculating cost for the cheaper 3.5 turbo model 
- Added validation to updating revoked key field

## 1.9.3 - 2024-02-13
### Added
- Added CORS support in the proxy

## 1.9.2 - 2024-02-06
### Fixed
- Fixed custom route tokens recording issue incurred by the new architecture

## 1.9.1 - 2024-02-06
### Fixed
- Fixed OpenAI chat completion endpoint being slow

## 1.9.0 - 2024-02-06
### Changed
- Drastically improved performance through event driven architecture

### Fixed
- Fixed API calls that exceeds cost limit not being blocked bug

## 1.8.2 - 2024-01-31
### Added
- Added support for new chat completion models
- Added new querying options for metrics and events API

## 1.8.1 - 2024-01-31
### Changed
- Extended default proxy request timeout to 10m

### Fixed
- Fixed streaming response stuck at context deadline exceeded error

## 1.8.0 - 2024-01-26
### Added
- Added key authentication for admin endpoints
  
## 1.7.6 - 2024-01-17
### Fixed
- Changed code to string in OpenAI error response

## 1.7.5 - 2024-01-17
### Fixed
- Fixed inability to parse OpenAI chat completion result
- Fixed problem with keys containing duplicated API credentials

## 1.7.4 - 2024-01-09
### Fixed
- Fixed key exceeding spend limit error code issue

## 1.7.3 - 2024-01-08
### Fixed
- Fixed a bug with route failover not working as expected

## 1.7.2 - 2024-01-01
### Added
- Fixed route caching issue
- Added default values to route step timeout and cache TTL

## 1.7.1 - 2023-12-29
### Added
- Started supporting caching in routes

## 1.7.0 - 2023-12-29
### Added
- Added the ability to make API calls with fallbacks
- Added the support for multiple sets of provider settings per key

## 1.6.0 - 2023-12-17
### Added
- Added support for Azure OpenAI embeddings and chat completions endpoints

## 1.5.1 - 2023-12-14
### Added
- Added support for minute spend limits

## 1.5.0 - 2023-12-14
### Added
- Added support for OpenAI audio endpoints
- Added support for OpenAI image endpoints
- Added support for monthly spend limits

### Fixed
- Removed deperacated fields when updating keys
- Fixed issues with inconsistent rate limit cache expiration date

## 1.4.1 - 2023-12-07
### Added
- Added path access control at the API key level
- Added model access control at the provider level
  
### Fixed
- Inability to update provider settings with only ```setting``` field

## 1.4.0 - 2023-12-05
### Added
- Added support for Anthropic completion endpoint
  
### Fixed
- Fixed streaming latency with custom provider and openai chat completion endpoints

## 1.3.1 - 2023-11-29
### Fixed
- Fixed bug with updating a custom provider

## 1.3.0 - 2023-11-29
### Added
- Added new API endpoints for custom providers

## 1.2.1 - 2023-11-27
### Added
- Added validation for changing setting when updating provider setting

### Changed
- Removed support for changing provider when updating provider setting

### Fixed
- Fixed issues with not being able to update provider setting name

## 1.2.0 - 2023-11-21
### Added
- Added new filters tags and provider to getting keys API 
- Added a new API endpoint for fetching an event
- Added a new API endpoint for fetching all provider settings
- Added more integration tests that cover critical paths

### Fixed
- Fixed issues with API key creation

## 1.1.1 - 2023-11-21
### Added
- Introduced new field called `name` to provider settings

### Fixed
- Fixed issues with PostgreSQL schema inconsistencies

## 1.1.0 - 2023-11-20
### Added
- Added support for OpenAI's embeddings API

## 1.0.4 - 2023-11-20
### Fixed
- Fixed configuration not found inconsistency with key and provider settings

### Changed
- Updated admin server to pull configurations every 5 sec by default
- Updated prod Dockerfile to have privacy mode turned on

### Added
- Added a default OpenAI proxy timeout of 180s
- Added model and keyId filters to the metrics API
- Added health check endpoints for both admin and proxy servers
- Started recording path and method for each proxy request

## 1.0.3 - 2023-11-09
### Fixed
- Fixed Datadog tags being in the wrong format

## 1.0.2 - 2023-11-09
### Added
- Added Datadog integration

### Fixed
- Fixed bug with proxy not fetching existing provider settings

## 1.0.1 - 2023-11-07
### Added
- Added support for gpt-4-turbo models in cost estimation

### Fixed
- Fixed proxy logic to prevent internal errors from forwarding requests to OpenAI
- Fixed proxy logic to prevent internal errors from receiving responses from OpenAI

## 1.0.0 - 2023-11-06
### Added
- Added a new API endpoint for creating and updating provider settings (e.g. OpenAI API key)

### Changed
- Removed having OpenAI API key as an required env vairable for running the proxy

### Breaking Changes
- Forced key creation to include a new field called settingId

## 0.0.9 - 2023-11-03
### Added
- Added a new API endpoint for retrieving metrics

## 0.0.8 - 2023-09-20
### Fixed
- Fixed open ai error response parsing

## 0.0.7 - 2023-09-03
### Added
- Added new reporting endpoint for retrieving the cumulative cost of an API key

## 0.0.6 - 2023-09-03
### Added
- Added support of cost estimation for fined tuned models
- Added logging for openAI error response

### Fixed
- Fixed incorrect logging format in production environment

## 0.0.5 - 2023-09-03
### Added
- Added logging for admin and proxy web servers

## 0.0.4 - 2023-08-30
### Fixed
- Fix PostgreSQL connection URI format issue

## 0.0.3 - 2023-08-30
### Fixed
- Updated PostgreSQL connection URI to be more versatile
- Changed env var POSTGRESQL_SSL_ENABLED to POSTGRESQL_SSL_MODE
- Updated proxy error response to exclude empty fields
### Added
- Added a new environment varible for configurting PostgreSQL DB Name

## 0.0.2 - 2023-08-29
### Fixed
- Fixed issue with rate limit
- Fixed issue with cost limit

## 0.0.1 - 2023-08-27
### Added
- Added a web server for managing key configurations
- Added a proxy server for openai request
- Added rate limit, cost limit and ttl to API keys

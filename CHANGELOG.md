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

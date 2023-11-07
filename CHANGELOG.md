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

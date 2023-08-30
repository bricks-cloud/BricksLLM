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
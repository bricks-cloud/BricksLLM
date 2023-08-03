## 0.0.1 - 2023-07-31
### Added
- Added a http web server for hosting `openai` prompts
- Added a `yaml` parser for reading bricksllm configurations
- Added support for [CORS](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS) configuration in the web server
- Added support for specifying `input` json schema
- Added support for `{input.field}` like syntax in the prompt template
- Added comprehensive logging in production and developer mode
- Added logger configuration for hiding sensitive fields using the `logger` field
- Added support for API key authenticaiton with `key_auth` field


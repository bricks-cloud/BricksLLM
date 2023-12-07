<p align="center">
<img src="./assets/bricks-logo.png" width="150" />
</p>

# **BricksLLM: AI Gateway For Putting LLM In Production**

<p align="center">
   <a href='https://www.ycombinator.com/'><img alt='YCombinator S22' src='https://img.shields.io/badge/Y%20Combinator-2022-orange'/></a>
   <a href='http://makeapullrequest.com'><img alt='PRs Welcome' src='https://img.shields.io/badge/PRs-welcome-43AF11.svg?style=shields'/></a>
   <a href="https://discord.gg/dFvdt4wqWh"><img src="https://img.shields.io/badge/discord-BricksLLM-blue?logo=discord&labelColor=2EB67D" alt="Join BricksLLM on Discord"></a>
   <a href="https://github.com/bricks-cloud/bricks/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-red" alt="License"></a>
</p>

**BricksLLM** is a cloud native AI gateway written in Go. Currently, it serves as a proxy to OpenAI. We let you create API keys that have rate limits, cost limits and TTLs. The API keys can be used in both development and production to achieve fine-grained access control that is not provided by OpenAI at the moment. The proxy is compatible with OpenAI API and its SDKs.

The vision of BricksLLM is to support many more large language models such as LLama2, Claude, PaLM2 etc, and streamline LLM operations.

## Roadmap
- [x] Access control via API key with rate limit, cost limit and ttl
- [x] Logging integration
- [x] Statsd integration
- [x] Custom Provider Integration
- [ ] PII detection and masking :construction:

## Getting Started
The easiest way to get started with BricksLLM is through [BricksLLM-Docker](https://github.com/bricks-cloud/BricksLLM-Docker).

### Step 1 - Clone BricksLLM-Docker repository
```bash
git clone https://github.com/bricks-cloud/BricksLLM-Docker
```

### Step 2 - Change to BricksLLM-Docker directory
```bash
cd BricksLLM-Docker
```

### Step 3 - Deploy BricksLLM locally with Postgresql and Redis
```bash
docker-compose up
```
You can run this in detach mode use the -d flag: `docker-compose up -d`


### Step 4 - Create a provider setting
```bash
curl -X PUT http://localhost:8001/api/provider-settings \
   -H "Content-Type: application/json" \
   -d '{
          "provider":"openai",
          "setting": {
             "apikey": "YOUR_OPENAI_KEY"
          }
      }'   
```
Copy the `id` from the response.

### Step 5 - Create a Bricks API key
Use `id` from the previous step as `settingId` to create a key with a rate limit of 2 req/min and a spend limit of 25 cents.
```bash
curl -X PUT http://localhost:8001/api/key-management/keys \
   -H "Content-Type: application/json" \
   -d '{
	      "name": "My Secret Key",
	      "key": "my-secret-key",
	      "tags": ["mykey"],
        "settingId": "ID_FROM_STEP_FOUR",
        "rateLimitOverTime": 2,
        "rateLimitUnit": "m",
        "costLimitInUsd": 0.25
      }'   
```

### Congradulations you are done!!!
Then, just redirect your requests to us and use OpenAI as you would normally. For example:
```bash
curl -X POST http://localhost:8002/api/providers/openai/v1/chat/completions \
   -H "Authorization: Bearer my-secret-key" \
   -H "Content-Type: application/json" \
   -d '{
          "model": "gpt-3.5-turbo",
          "messages": [
              {
                  "role": "system",
                  "content": "hi"
              }
          ]
      }'
```

Or if you're using an SDK, you could change its `baseURL` to point to us. For example:
```js
// OpenAI Node SDK v4
import OpenAI from 'openai';

const openai = new OpenAI({
  apiKey: "some-secret-key", // key created earlier
  baseURL: "http://localhost:8002/api/providers/openai/v1", // redirect to us
});
```

## How to Update?
For updating to the latest version
```bash
docker pull luyuanxin1995/bricksllm:latest
```

For updating to a particular version
```bash
docker pull luyuanxin1995/bricksllm:1.4.0
```

# Documentation
## Environment variables
> | Name | type | description | default |
> |---------------|-----------------------------------|----------|-|
> | `POSTGRESQL_HOSTS`       | required | Hosts for Postgresql DB. Seperated by , | `localhost` |
> | `POSTGRESQL_DB_NAME`       | optional | Name for Postgresql DB. |
> | `POSTGRESQL_USERNAME`         | required | Postgresql DB username |
> | `POSTGRESQL_PASSWORD`         | required | Postgresql DB password |
> | `POSTGRESQL_SSL_MODE`         | optional | Postgresql SSL mode| `disable`
> | `POSTGRESQL_PORT`         | optional | The port that Postgresql DB runs on| `5432`
> | `POSTGRESQL_READ_TIME_OUT`         | optional | Timeout for Postgresql read operations | `2s`
> | `POSTGRESQL_WRITE_TIME_OUT`         | optional | Timeout for Postgresql write operations | `1s`
> | `REDIS_HOSTS`         | required | Host for Redis. Seperated by , | `localhost`
> | `REDIS_PASSWORD`         | optional | Redis Password |
> | `REDIS_PORT`         | optional | The port that Redis DB runs on | `6379`
> | `REDIS_READ_TIME_OUT`         | optional | Timeout for Redis read operations | `1s`
> | `REDIS_WRITE_TIME_OUT`         | optional | Timeout for Redis write operations | `500ms`
> | `IN_MEMORY_DB_UPDATE_INTERVAL`         | optional | The interval BricksLLM API gateway polls Postgresql DB for latest key configurations | `1s`
> | `STATS_PROVIDER`         | optional | This value can only be datadog. Required for integration with Datadog.  |
> | `PROXY_TIMEOUT`         | optional | This value can only be datadog. Required for integration with Datadog.  |

## Configuration Endpoints
The configuration server runs on Port `8001`.
<details>
  <summary>Get keys: <code>GET</code> <code><b>/api/key-management/keys</b></code></summary>

##### Description
This endpoint is set up for retrieving key configurations using a query param called tag.

##### Query Parameters

> | name   |  type      | data type      | description                                          |
> |--------|------------|----------------|------------------------------------------------------|
> | `tag` |  optional   | string         | Identifier attached to a key configuration                  |
> | `tags` |  optional  | array of string         | Identifiers attached to a key configuration                  |
> | `provider` |  optional  | string         | Provider attached to a key provider configuration. Its value can only be `openai`.                 |

##### Error Response

> | http code     | content-type                      |
> |---------------|-----------------------------------|
> | `400`, `500`         | `application/json`                |

> | Field     | type | example                      |
> |---------------|-----------------------------------|-|
> | status         | `int` | 400            |
> | title         | `string` | request body reader error             |
> | type         | `string` | /errors/request-body-read             |
> | detail         | `string` | something is wrong            |
> | instance         | `string` | /api/key-management/keys            |

##### Response

> | Response Body |
> |---------------|
> | `[]KeyConfiguration` |

Fields of KeyConfiguration
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | name | `string` | spike's developer key | Name of the API key. |
> | createdAt | `int64` | 1257894000 | Key configuration creation time in unix.  |
> | updatedAt | `int64` | 1257894000 | Key configuration update time in unix.  |
> | revoked | `boolean` | true | Indicator for whether the key is revoked.  |
> | revokedReason | `string` | The key has expired | Reason for why the key is revoked.  |
> | tags | `[]string` | ["org-tag-12345"]             | Identifiers associated with the key. |
> | keyId | `string` | 550e8400-e29b-41d4-a716-446655440000 | Unique identifier for the key.  |
> | costLimitInUsd | `float64` | `5.5` | Total spend limit of the API key.
> | costLimitInUsdOverTime | `float64` | `2` | Total spend within period of time. This field is required if costLimitInUsdUnit is specified.   |
> | costLimitInUsdUnit | `enum` | d                       | Time unit for costLimitInUsdOverTime. Possible values are [`h`, `m`, `s`, `d`].      |
> | rateLimitOverTime | `int` | `2` | rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitUnit | `string` | m                         |  Time unit for rateLimitOverTime. Possible values are [`h`, `m`, `s`, `d`]       |
> | ttl | `string` | 2d | time to live. Available units are [`s`, `m`, `h`] |

</details>


<details>
  <summary>Create key: <code>PUT</code> <code><b>/api/key-management/keys</b></code></summary>

##### Description
This endpoint is set up for retrieving key configurations using a query param called tag.

##### Request
```
PathConfig
```
> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | path | required | `string` | /api/providers/openai/v1/chat/completion | Allowed path |
> | method | required | `string` | POST | HTTP Method


> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | name | required | `string` | spike's developer key | Name of the API key. |
> | tags | optional | `[]string` | `["org-tag-12345"] `            | Identifiers associated with the key. |
> | key | required | `string` | abcdef12345 | API key |
> | settingId | required | `string` | 98daa3ae-961d-4253-bf6a-322a32fdca3d | API key |
> | costLimitInUsd | optional | `float64` | `5.5` | Total spend limit of the API key.
> | costLimitInUsdOverTime | optional | `float64` | `2` | Total spend within period of time. This field is required if costLimitInUsdUnit is specified.   |
> | costLimitInUsdUnit | optional | `enum` | d                       | Time unit for costLimitInUsdOverTime. Possible values are [`h`, `d`].      |
> | rateLimitOverTime | optional | `int` | 2 | rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitUnit | optional | `enum` | m                         |  Time unit for rateLimitOverTime. Possible values are [`h`, `m`, `s`, `d`]       |
> | ttl | optional | `string` | 2d | time to live. Available units are [`s`, `m`, `h`] |
> | allowedPaths | optional | `[]PathConfig` | 2d | Pathes allowed for access |


##### Error Response

> | http code     | content-type                      |
> |---------------|-----------------------------------|
> | `400`, `500`         | `application/json`                |

> | Field     | type | example                      |
> |---------------|-----------------------------------|-|
> | status         | `int` | `400`            |
> | title         | `string` | request body reader error             |
> | type         | `string` | /errors/request-body-read             |
> | detail         | `string` | something is wrong            |
> | instance         | `string` | /api/key-management/keys            |

##### Responses
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | name | `string` | spike's developer key | Name of the API key. |
> | createdAt | `int64` | `1257894000` | Key configuration creation time in unix.  |
> | updatedAt | `int64` | `1257894000` | Key configuration update time in unix.  |
> | revoked | `boolean` | `true` | Indicator for whether the key is revoked.  |
> | revokedReason | `string` | The key has expired | Reason for why the key is revoked.  |
> | tags | `[]string` | ["org-tag-12345"]             | Identifiers associated with the key. |
> | keyId | `string` | 550e8400-e29b-41d4-a716-446655440000 | Unique identifier for the key.  |
> | costLimitInUsd | `float64` | `5.5` | Total spend limit of the API key.
> | costLimitInUsdOverTime | `float64` | 2 | Total spend within period of time. This field is required if costLimitInUsdUnit is specified.   |
> | costLimitInUsdUnit | `enum` | d                       | Time unit for costLimitInUsdOverTime. Possible values are [`h`, `d`].      |
> | rateLimitOverTime | `int` | `2` | rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitOverTime | `int` | `2` | rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitUnit | `string` | m                         |  Time unit for rateLimitOverTime. Possible values are [`h`, `m`, `s`, `d`].       |
> | ttl | `string` | 2d | time to live. Available units are [`s`, `m`, `h`] |
> | allowedPaths | `[]PathConfig` | `[{ "path": "/api/providers/openai/v1/chat/completion", method: "POST"}]` | Allowed paths that can be accessed using the key. |

</details>

<details>
  <summary>Update key: <code>PATCH</code> <code><b>/api/key-management/keys/{keyId}</b></code></summary>

##### Description
This endpoint is set up for updating key configurations using key id.

##### Parameters
> | name   |  type      | data type      | description                                          |
> |--------|------------|----------------|------------------------------------------------------|
> | `keyId` |  required  | string         | Unique key configuration identifier.                  |

##### Request
```
PathConfig
```
> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | path | required | `string` | /api/providers/openai/v1/chat/completion | Allowed path |
> | method | required | `string` | POST | HTTP Method

> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | name | optional | `string` | spike's developer key | Name of the API key. |
> | tags | optional | `[]string` | `["org-tag-12345"]`             | Identifiers associated with the key. |
> | revoked | optional |  `boolean` | `true` | Indicator for whether the key is revoked.  |
> | revokedReason| optional | `string` | The key has expired | Reason for why the key is revoked.  |
> | costLimitInUsdOverTime | optional | `float64` | `2` | Total spend within period of time. This field is required if costLimitInUsdUnit is specified.   |
> | costLimitInUsdUnit | optional | `enum` | d                       | Time unit for costLimitInUsdOverTime. Possible values are [`h`, `d`].      |
> | rateLimitOverTime | optional | `int` | `2` | rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitUnit | optional | `enum` | m                         |  Time unit for rateLimitOverTime. Possible values are [`h`, `m`, `s`, `d`]       |
> | allowedPaths | optional | `[]PathConfig` | 2d | Pathes allowed for access. |

##### Error Response

> | http code     | content-type                      |
> |---------------|-----------------------------------|
> | `400`, `500`         | `application/json`                |

> | Field     | type | example                      |
> |---------------|-----------------------------------|-|
> | status         | `int` | 400            |
> | title         | `string` | request body reader error             |
> | type         | `string` | /errors/request-body-read             |
> | detail         | `string` | something is wrong            |
> | instance         | `string` | /api/key-management/keys            |

##### Response
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | name | `string` | spike's developer key | Name of the API key. |
> | createdAt | `int64` | `1257894000` | Key configuration creation time in unix.  |
> | updatedAt | `int64` | `1257894000` | Key configuration update time in unix.  |
> | revoked | `boolean` | `true` | Indicator for whether the key is revoked.  |
> | revokedReason | `string` | The key has expired | Reason for why the key is revoked.  |
> | tags | `[]string` | `["org-tag-12345"]`           | Identifiers associated with the key. |
> | keyId | `string` | 550e8400-e29b-41d4-a716-446655440000 | Unique identifier for the key.  |
> | costLimitInUsd | `float64` | `5.5` | Total spend limit of the API key.
> | costLimitInUsdOverTime | `float64` | `2` | Total spend within period of time. This field is required if costLimitInUsdUnit is specified.   |
> | costLimitInUsdUnit | `enum` | d                       | Time unit for costLimitInUsdOverTime. Possible values are [`h`, `d`].      |
> | rateLimitOverTime | `int` | `2` | rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitUnit | `string` | m                         |  Time unit for rateLimitOverTime. Possible values are [`h`, `m`, `s`, `d`]       |
> | ttl | `string` | 2d | time to live. Available units are [`s`, `m`, `h`] |
> | allowedPaths | `[]PathConfig` | `[{ "path": "/api/providers/openai/v1/chat/completion", method: "POST"}]` | Allowed paths that can be accessed using the key. |

</details>

<details>
  <summary>Create a provider setting: <code>POST</code> <code><b>/api/provider-settings</b></code></summary>

##### Description
This endpoint is creating a provider setting.

##### Request
> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | provider | required | `enum` | openai | This value can only be `openai` as for now. |
> | setting | required | `object` | `{ "apikey": "YOUR_OPENAI_KEY" }`            | A map of values used for authenticating with the selected provider. |
> | setting.apikey | required | `string` | xx-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx | This field is required if `provider` is `openai`. |
> | name | optional | `string` | YOUR_PROVIDER_SETTING_NAME | This field is used for giving a name to provider setting |
> | allowedModels | `[]string` | `["text-embedding-ada-002"]` | Allowed models for this provider setting. |


##### Error Response
> | http code     | content-type                      |
> |---------------|-----------------------------------|
> | `400`, `500`         | `application/json`                |

> | Field     | type | example                      |
> |---------------|-----------------------------------|-|
> | status         | `int` | `400`            |
> | title         | `string` | request body reader error             |
> | type         | `string` | /errors/request-body-read             |
> | detail         | `string` | something is wrong            |
> | instance         | `string` | /api/provider-settings           |

##### Response
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | createdAt | `int64` | `1699933571` | Unix timestamp for creation time.  |
> | updatedAt | `int64` | `1699933571` | Unix timestamp for update time. |
> | provider | `enum` | `openai` | This value can only be `openai` as for now. |
> | id | `string` | 98daa3ae-961d-4253-bf6a-322a32fdca3d | This value is a unique identifier. |
> | name | `string` | YOUR_PROVIDER_SETTING_NAME | Provider setting name. |
> | allowedModels | `[]string` | `["text-embedding-ada-002"]` | Allowed models for this provider setting. |

</details>


<details>
  <summary>Get all provider settings: <code>GET</code> <code><b>/api/provider-settings</b></code></summary>

##### Description
This endpoint is getting all provider settings.

##### Request
> | Field | type | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|


##### Error Response
> | http code     | content-type                      |
> |---------------|-----------------------------------|
> | `500`        | `application/json`                |

> | Field     | type | example                      |
> |---------------|-----------------------------------|-|
> | status         | `int` | `400`            |
> | title         | `string` | request body reader error             |
> | type         | `string` | /errors/request-body-read             |
> | detail         | `string` | something is wrong            |
> | instance         | `string` | /api/provider-settings           |

##### Response
```
[]ProviderSetting
```

ProviderSetting
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | createdAt | `int64` | `1699933571` | Unix timestamp for creation time.  |
> | updatedAt | `int64` | `1699933571` | Unix timestamp for update time. |
> | provider | `enum` | `openai` | This value can only be `openai` as for now. |
> | id | `string` | 98daa3ae-961d-4253-bf6a-322a32fdca3d | This value is a unique identifier. |
> | name | `string` | YOUR_PROVIDER_SETTING_NAME | Provider setting name. |
> | allowedModels | `[]string` | `["text-embedding-ada-002"]` | Allowed models for this provider setting. |

</details>

<details>
  <summary>Update a provider setting: <code>PATCH</code> <code><b>/api/provider-settings/:id</b></code></summary>

##### Description
This endpoint is updating a provider setting .

##### Parameters
> | name   |  type      | data type      | description                                          |
> |--------|------------|----------------|------------------------------------------------------|
> | `id` |  required  | `string`         | Unique identifier for the provider setting that you want to update.                  |

##### Request
> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | setting | required | `object` | `{ "apikey": "YOUR_OPENAI_KEY" }`            | A map of values used for authenticating with the selected provider. |
> | setting.apikey | required | `string` | xx-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx | This field is required if `provider` is `openai`. |
> | name | optional | `string` | YOUR_PROVIDER_SETTING_NAME | This field is used for giving a name to provider setting |
> | allowedModels | `[]string` | `["text-embedding-ada-002"]` | Allowed models for this provider setting. |

##### Error Response

> | http code     | content-type                      |
> |---------------|-----------------------------------|
> | `400`, `500`         | `application/json`                |

> | Field     | type | example                      |
> |---------------|-----------------------------------|-|
> | status         | `int` | `400`            |
> | title         | `string` | request body reader error             |
> | type         | `string` | /errors/request-body-read             |
> | detail         | `string` | something is wrong            |
> | instance         | `string` | /api/provider-settings           |

##### Response
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | createdAt | `int64` | `1699933571` | Unix timestamp for creation time.  |
> | updatedAt | `int64` | `1699933571` | Unix timestamp for update time. |
> | provider | `enum` | `openai` | This value can only be `openai` as for now. |
> | id | `string` | `98daa3ae-961d-4253-bf6a-322a32fdca3d` | This value is a unique identifier |
> | name | `string` | YOUR_PROVIDER_SETTING_NAME | Provider setting name. |
> | allowedModels | `[]string` | `["text-embedding-ada-002"]` | Allowed models for this provider setting. |

</details>

<details>
  <summary>Retrieve Metrics: <code>POST</code> <code><b>/api/reporting/events</b></code></summary>

##### Description
This endpoint is retrieving aggregated metrics given an array of key ids and tags.

##### Request
> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | keyIds | required | `[]string` | `["key-1", "key-2", "key-3" ]` | Array of ids that specicify the keys that you want to aggregate stats from. |
> | tags | required | `[]string` | `["tag-1", "tag-2"]`           | Array of tags that specicify the keys that you want to aggregate stats from. |
> | start | required | `int64` | `1699933571` | Start timestamp for the requested timeseries data. |
> | end | required | `int64` | `1699933571` | End timestamp for the requested timeseries data. |
> | increment | required | `int` | `60` | This field is the increment in seconds for the requested timeseries data. |

##### Error Response
> | http code     | content-type                      |
> |---------------|-----------------------------------|
> | `500`         | `application/json`                |

> | Field     | type | example                      |
> |---------------|-----------------------------------|-|
> | status         | `int` | `400`            |
> | title         | `string` | request body reader error             |
> | type         | `string` | /errors/request-body-read             |
> | detail         | `string` | something is wrong            |
> | instance         | `string` | /api/provider-settings           |

##### Response
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | dataPoints | `[]dataPoint` | `[{ "timeStamp": 1699933571, "numberOfRequests": 1, "costInUsd": 0.8, "latencyInMs": 600, "promptTokenCount": 0, "completionTokenCount": 0, "successCount": 1 }]` | Unix timestamp for creation time.  |
> | latencyInMsMedian | `float64` | `656.7` | Median latency for the given time period. |
> | latencyInMs99th | `float64` | `555.7` | 99th percentile latency for the given time period. |
> | dataPoints.[].timeStamp | `int64` | `555.7` | Timestamp of the data point |
> | dataPoints.[].numberOfRequests | `int64` | `555.7` | Aggregated number of http requests over the given time increment. |
> | dataPoints.[].costInUsd | `int64` | `555.7` | Aggregated cost of http requests over the given time increment. |
> | dataPoints.[].latencyInMs | `float64` | `555.7` | Aggregated latency of http requests over the given time increment. |
> | dataPoints.[].promptTokenCount | `int` | `555.7` | Aggregated prompt token counts over the given time increment. |
> | dataPoints.[].completionTokenCount | `int` | `555.7` | Aggregated completion token counts over the given time increment. |
> | dataPoints.[].successCount | `int` | `555.7` | Aggregated number of successful http requests over the given time increment. |

</details>

<details>
  <summary>Get events: <code>GET</code> <code><b>/api/events</b></code></summary>

##### Description
This endpoint is for getting events.

##### Query Parameters
> | name   |  type      | data type      | description                                          |
> |--------|------------|----------------|------------------------------------------------------|
> | `customId` |  optional   | string         | Custom identifier attached to an event                  |

##### Error Response
> | http code     | content-type                      |
> |---------------|-----------------------------------|
> | `500`        | `application/json`                |

> | Field     | type | example                      |
> |---------------|-----------------------------------|-|
> | status         | `int` | `400`            |
> | title         | `string` | request body reader error             |
> | type         | `string` | /errors/request-body-read             |
> | detail         | `string` | something is wrong            |
> | instance         | `string` | /api/provider-settings           |

##### Response
```
[]Event
```

Event
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | id | `int64` | `1699933571` | Unique identifier associated with the event.  |
> | created_at | `int64` | `1699933571` | Unix timestamp for creation time.  |
> | tags | `int64` | `["YOUR_TAG"]` | Tags of the key. |
> | key_id | `string` | YOUR_KEY_ID | Key Id associated with the proxy request. |
> | cost_in_usd | `float64` | 0.0004 | Cost incured by the proxy request. |
> | model | `string` | gpt-4-1105-preview | Model used in the proxy request. |
> | provider | `string` | `openai` | Provider for the proxy request. |
> | status | `int` | `200` | Http status. |
> | prompt_token_count | `int` | `8` | Prompt token count of the proxy request. |
> | completion_token_count | `int` | `16` | Completion token counts of the proxy request. |
> | latency_in_ms | `int` | `160` | Provider setting name. |
> | path | `string` | /api/v1/chat/completion | Provider setting name. |
> | method | `string` | POST | Http method for the assoicated proxu request. |
> | custom_id | `string` | YOUR_CUSTOM_ID | Custom Id passed by the user in the headers of proxy requests. |
</details>

<details>
  <summary>Create custom provider: <code>POST</code> <code><b>/api/custom/providers</b></code></summary>

##### Description
This endpoint is creating custom providers.

##### Route Config
> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | path | required | `string` | `/chat/completion` | Path associated with the custom provider route. It must be unique within the custom provider. |
> | target_url | required | `string` | `https://api.openai.com/v1/chat/completions` | Proxy destination URL for the custom provider route. |
> | model_location | required | `string` | `model` | JSON field for the model in the HTTP request. |
> | request_prompt_location | required | `string` | `messages.#.content` | JSON field for the prompt request in the HTTP request. |
> | response_completion_location | required | `string` | `choices.#.message.content` | JSON field for the completion content in the HTTP response. |
> | stream_location | required | `string` | `stream` | JSON field for the stream boolean in the HTTP request. |
> | stream_end_word | required | `string` | `[DONE]` | End word for the stream. |
> | stream_response_completion_location | required | `string` | `choices.#.delta.content` | JSON field for the completion content in the streaming response. |
> | stream_max_empty_messages | required | `int` | `10` | Number of max empty messages in stream. |


##### Request
> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | provider | required | `string` | `bricks`  | Unique identifier associated with the route config. |
> | route_configs | required | `[]RouteConfig` | `{{ "path": "/chat/completions", "target_url": "https://api.openai.com/v1/chat/completions" }}` | Route configurations for the custom provider. |
> | authentication_param | optional | `string` | `apikey` | The authentication parameter required for. |

##### Error Response
> | http code     | content-type                      |
> |---------------|-----------------------------------|
> | `500`, `400`        | `application/json`                |

> | Field     | type | example                      |
> |---------------|-----------------------------------|-|
> | status         | `int` | `400`            |
> | title         | `string` | request body reader error             |
> | type         | `string` | /errors/request-body-read             |
> | detail         | `string` | something is wrong            |
> | instance         | `string` | /api/custom/providers           |

##### Response
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | id | `int64` | `1699933571` | Unique identifier associated with the event.  |
> | created_at | `int64` | `1699933571` | Unix timestamp for creation time.  |
> | updated_at | `int64` | `1699933571` | Unix timestamp for update time.  |
> | provider | `string` | `bricks`  | Unique identifier associated with the route config. |
> | route_configs | `[]RouteConfig` | `{{ "path": "/chat/completions", "target_url": "https://api.openai.com/v1/chat/completions" }}` | Start timestamp for the requested timeseries data. |
> | authentication_param | `string` | `apikey` | The authentication parameter required for. |
</details>

<details>
  <summary>Update custom provider: <code>PATCH</code> <code><b>/api/custom/providers/:id</b></code></summary>

##### Description
This endpoint is updating a custom provider.

##### Route Config
> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | path | required | `string` | `/chat/completion` | Path associated with the custom provider route. It must be unique within the custom provider. |
> | target_url | required | `string` | `https://api.openai.com/v1/chat/completions` | Proxy destination URL for the custom provider route. |
> | model_location | required | `string` | `model` | JSON field for the model in the HTTP request. |
> | request_prompt_location | required | `string` | `messages.#.content` | JSON field for the prompt request in the HTTP request. |
> | response_completion_location | required | `string` | `choices.#.message.content` | JSON field for the completion content in the HTTP response. |
> | stream_location | required | `string` | `stream` | JSON field for the stream boolean in the HTTP request. |
> | stream_end_word | required | `string` | `[DONE]` | End word for the stream. |
> | stream_response_completion_location | required | `string` | `choices.#.delta.content` | JSON field for the completion content in the streaming response. |
> | stream_max_empty_messages | required | `int` | `10` | Number of max empty messages in stream. |


##### Request
> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | route_configs | optional | `[]RouteConfig` | `{{ "path": "/chat/completions", "target_url": "https://api.openai.com/v1/chat/completions" }}` | Route configurations for the custom provider. |
> | authentication_param | optional | `string` | `apikey` | The authentication parameter required for. |

##### Error Response
> | http code     | content-type                      |
> |---------------|-----------------------------------|
> | `500`, `404`, `400`        | `application/json`                |

> | Field     | type | example                      |
> |---------------|-----------------------------------|-|
> | status         | `int` | `400`            |
> | title         | `string` | request body reader error             |
> | type         | `string` | /errors/request-body-read             |
> | detail         | `string` | something is wrong            |
> | instance         | `string` | /api/custom/providers           |

##### Response
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | id | `int64` | `1699933571` | Unique identifier associated with the event.  |
> | created_at | `int64` | `1699933571` | Unix timestamp for creation time.  |
> | updated_at | `int64` | `1699933571` | Unix timestamp for update time.  |
> | provider | `string` | `bricks`  | Unique identifier associated with the route config. |
> | route_configs | `[]RouteConfig` | `{{ "path": "/chat/completions", "target_url": "https://api.openai.com/v1/chat/completions" }}` | Start timestamp for the requested timeseries data. |
> | authentication_param | `string` | `apikey` | The authentication parameter required for. |
</details>

<details>
  <summary>Get custom providers: <code>GET</code> <code><b>/api/custom/providers</b></code></summary>

##### Description
This endpoint is for getting custom providers.

##### Route Config
> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | path | required | `string` | `/chat/completion` | Path associated with the custom provider route. It must be unique within the custom provider. |
> | target_url | required | `string` | `https://api.openai.com/v1/chat/completions` | Proxy destination URL for the custom provider route. |
> | model_location | required | `string` | `model` | JSON field for the model in the HTTP request. |
> | request_prompt_location | required | `string` | `messages.#.content` | JSON field for the prompt request in the HTTP request. |
> | response_completion_location | required | `string` | `choices.#.message.content` | JSON field for the completion content in the HTTP response. |
> | stream_location | required | `string` | `stream` | JSON field for the stream boolean in the HTTP request. |
> | stream_end_word | required | `string` | `[DONE]` | End word for the stream. |
> | stream_response_completion_location | required | `string` | `choices.#.delta.content` | JSON field for the completion content in the streaming response. |
> | stream_max_empty_messages | required | `int` | `10` | Number of max empty messages in stream. |


##### Request
> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | route_configs | optional | `[]RouteConfig` | `{{ "path": "/chat/completions", "target_url": "https://api.openai.com/v1/chat/completions" }}` | Route configurations for the custom provider. |
> | authentication_param | optional | `string` | `apikey` | The authentication parameter required for. |

##### Error Response
> | http code     | content-type                      |
> |---------------|-----------------------------------|
> | `500`        | `application/json`                |

> | Field     | type | example                      |
> |---------------|-----------------------------------|-|
> | status         | `int` | `400`            |
> | title         | `string` | request body reader error             |
> | type         | `string` | /errors/request-body-read             |
> | detail         | `string` | something is wrong            |
> | instance         | `string` | /api/custom/providers           |

##### Response
```
[]Provider
```

Provider
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | id | `int64` | `1699933571` | Unique identifier associated with the event.  |
> | created_at | `int64` | `1699933571` | Unix timestamp for creation time.  |
> | updated_at | `int64` | `1699933571` | Unix timestamp for update time.  |
> | provider | `string` | `bricks`  | Unique identifier associated with the route config. |
> | route_configs | `[]RouteConfig` | `{{ "path": "/chat/completions", "target_url": "https://api.openai.com/v1/chat/completions" }}` | Start timestamp for the requested timeseries data. |
> | authentication_param | `string` | `apikey` | The authentication parameter required for. |
</details>

## OpenAI Proxy
The OpenAI proxy runs on Port `8002`.

##### Headers
> | name   |  type      | data type      | description                                          |
> |--------|------------|----------------|------------------------------------------------------|
> | `x-custom-event-id` |  optional  | `string`         | Custom Id that can be used to retrieve an event associated with each proxy request.

### Chat Completion
<details>
  <summary>Call OpenAI chat completions: <code>POST</code> <code><b>/api/providers/openai/v1/chat/completions</b></code></summary>

##### Description
This endpoint is set up for proxying OpenAI chat completion requests. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/chat).

</details>


### Embeddings
<details>
  <summary>Call OpenAI embeddings: <code>POST</code> <code><b>/api/providers/openai/v1/embeddings</b></code></summary>

##### Description
This endpoint is set up for proxying OpenAI embedding requests. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/embeddings/create).

</details>

### Moderations

<details>
  <summary>Call OpenAI moderations: <code>POST</code> <code><b>/api/providers/openai/v1/moderations</b></code></summary>

##### Description
This endpoint is set up for proxying OpenAI moderation requests. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/moderations/create).

</details>

### Models
<details>
  <summary>Get OpenAI models: <code>GET</code> <code><b>/api/providers/openai/v1/models</b></code></summary>

##### Description
This endpoint is set up for retrieving OpenAI models. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/models/list).

</details>

<details>
  <summary>Retrieve an OpenAI model: <code>GET</code> <code><b>/api/providers/openai/v1/models/:model</b></code></summary>

##### Description
This endpoint is set up for retrieving an OpenAI model. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/models/retrieve).

</details>

### Files
<details>
  <summary>List files: <code>GET</code> <code><b>/api/providers/openai/v1/files</b></code></summary>

##### Description
This endpoint is set up for list OpenAI files. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/files/list).

</details>

<details>
  <summary>Upload a file: <code>POST</code> <code><b>/api/providers/openai/v1/files</b></code></summary>

##### Description
This endpoint is set up for creating an OpenAI file. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/files/create).

</details>


<details>
  <summary>Delete a file: <code>POST</code> <code><b>/api/providers/openai/v1/files/:file_id</b></code></summary>

##### Description
This endpoint is set up for creating an OpenAI file. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/files/delete).

</details>

<details>
  <summary>Retrieve a file: <code>GET</code> <code><b>/api/providers/openai/v1/files/:file_id</b></code></summary>

##### Description
This endpoint is set up for retrieving an OpenAI file. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/files/retrieve).

</details>

<details>
  <summary>Retrieve file content: <code>GET</code> <code><b>/api/providers/openai/v1/files/:file_id/content</b></code></summary>

##### Description
This endpoint is set up for retrieving an OpenAI file content. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/files/retrieve-contents).

</details>

## Anthropic Proxy
The custom provider proxy runs on Port `8002`.

<details>
  <summary>Create Anthropic completion: <code>POST</code> <code><b>/api/providers/anthropic/v1/complete</b></code></summary>

##### Description
This endpoint is set up for proxying Anthropic completion requests. Documentation for this endpoint can be found [here](https://docs.anthropic.com/claude/reference/complete_post).

</details>

## Custom Provider Proxy
The custom provider proxy runs on Port `8002`.

<details>
  <summary>Call custom providers: <code>POST</code> <code><b>/api/custom/providers/:provider/*</b></code></summary>

##### Description
First you need to use create custom providers endpoint to create custom providers. Then create corresponding provider setting for the newly created custom provider. Afterward, you can start creating keys associated with the custom provider, and use the keys to access this endpoint by placing the created key in ```Authorization: Bearer YOUR_BRICKSLLM_KEY``` as part of your HTTP request headers.

</details>
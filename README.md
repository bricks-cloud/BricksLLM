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

> [!TIP]
> A [managed version of BricksLLM](https://www.trybricks.ai?utm_source=github&utm_medium=repo&utm_campaign=bricksllm) is also available! It is production ready, and comes with a dashboard to make interacting with BricksLLM easier. Try us out for free today!

**BricksLLM** is a cloud native AI gateway written in Go. Currently, it provides native support for OpenAI, Anthropic, Azure OpenAI and vLLM. BricksLLM aims to provide enterprise level infrastructure that can power any LLM production use cases. Here are some use cases for BricksLLM: 

* Set LLM usage limits for users on different pricing tiers
* Track LLM usage on a per user and per organization basis
* Block or redact requests containing PIIs
* Improve LLM reliability with failovers, retries and caching
* Distribute API keys with rate limits and cost limits for internal development/production use cases
* Distribute API keys with rate limits and cost limits for students

## Features
- [x] [PII detection and masking](https://github.com/bricks-cloud/BricksLLM/blob/main/cookbook/pii_detection.md)
- [x] Rate limit
- [x] Cost control
- [x] Cost analytics
- [x] Request analytics
- [x] [Caching](https://github.com/bricks-cloud/BricksLLM/blob/main/cookbook/openai_with_azure_openai_failover.md)
- [x] [Request Retries](https://github.com/bricks-cloud/BricksLLM/blob/main/cookbook/openai_with_azure_openai_failover.md)
- [x] [Failover](https://github.com/bricks-cloud/BricksLLM/blob/main/cookbook/openai_with_azure_openai_failover.md)
- [x] [Model access control](https://github.com/bricks-cloud/BricksLLM/blob/main/cookbook/granular_access_control.md)
- [x] [Endpoint access control](https://github.com/bricks-cloud/BricksLLM/blob/main/cookbook/granular_access_control.md)
- [x] Native support for all OpenAI endpoints
- [x] Native support for Anthropic
- [x] Native support for Azure OpenAI
- [x] [Native support for vLLM](https://github.com/bricks-cloud/BricksLLM/blob/main/cookbook/vllm_integration.md)
- [x] Native support for Deepinfra
- [x] Support for custom deployments
- [x] Integration with custom models
- [x] Datadog integration
- [x] Logging with privacy control


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
docker compose up
```
You can run this in detach mode use the -d flag: `docker compose up -d`


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
        "settingIds": ["ID_FROM_STEP_FOUR"],
        "rateLimitOverTime": 2,
        "rateLimitUnit": "m",
        "costLimitInUsd": 0.25
      }'   
```

### Congratulations you are done!!!
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
> | `POSTGRESQL_HOSTS`       | required | Hosts for Postgresql DB. Separated by , | `localhost` |
> | `POSTGRESQL_DB_NAME`       | optional | Name for Postgresql DB. |
> | `POSTGRESQL_USERNAME`         | required | Postgresql DB username |
> | `POSTGRESQL_PASSWORD`         | required | Postgresql DB password |
> | `POSTGRESQL_SSL_MODE`         | optional | Postgresql SSL mode| `disable`
> | `POSTGRESQL_PORT`         | optional | The port that Postgresql DB runs on| `5432`
> | `POSTGRESQL_READ_TIME_OUT`         | optional | Timeout for Postgresql read operations | `2m`
> | `POSTGRESQL_WRITE_TIME_OUT`         | optional | Timeout for Postgresql write operations | `5s`
> | `REDIS_HOSTS`         | required | Host for Redis. Separated by , | `localhost`
> | `REDIS_PASSWORD`         | optional | Redis Password |
> | `REDIS_PORT`         | optional | The port that Redis DB runs on | `6379`
> | `REDIS_READ_TIME_OUT`         | optional | Timeout for Redis read operations | `1s`
> | `REDIS_WRITE_TIME_OUT`         | optional | Timeout for Redis write operations | `500ms`
> | `IN_MEMORY_DB_UPDATE_INTERVAL`         | optional | The interval BricksLLM API gateway polls Postgresql DB for latest key configurations | `1s`
> | `STATS_PROVIDER`         | optional | This value can only be datadog. Required for integration with Datadog.  |
> | `PROXY_TIMEOUT`         | optional | Timeout for proxy HTTP requests. | `600s` |
> | `NUMBER_OF_EVENT_MESSAGE_CONSUMERS`         | optional | Number of event message consumers that help handle counting tokens and inserting event into db.  | `3` |
> | `AWS_SECRET_ACCESS_KEY`         | optional | It is for PII detection feature.  | `5s` |
> | `AWS_ACCESS_KEY_ID`         | optional | It is for using PII detection feature.  | `5s` |
> | `AMAZON_REGION`         | optional | Region for AWS.  | `us-west-2` |
> | `AMAZON_REQUEST_TIMEOUT`         | optional | Timeout for amazon requests.  | `5s` |
> | `AMAZON_CONNECTION_TIMEOUT`         | optional | Timeout for amazon connection.  | `10s` |
> | `AWS_SECRET_ACCESS_KEY`         | optional | Required for PII detection features.  |
> | `AWS_ACCESS_KEY_ID`         | optional | Required for PII detection features.  |

## Configuration Endpoints
The configuration server runs on Port `8001`.

##### Headers
> | name   |  type      | data type      | description                                          |
> |--------|------------|----------------|------------------------------------------------------|
> | `X-API-KEY` |  optional  | `string`         | Key authentication header.


<details>
  <summary>Get keys: <code>GET</code> <code><b>/api/key-management/keys</b></code></summary>

##### Description
This endpoint is set up for retrieving key configurations using a query param called tag.

##### Query Parameters
> | name   |  type      | data type      | description                                          |
> |--------|------------|----------------|------------------------------------------------------|
> | `tag` |  optional   | `string`         | Identifier attached to a key configuration                  |
> | `tags` |  optional  | `[]string`         | Identifiers attached to a key configuration                  |
> | `provider` |  optional  | `string`         | Provider attached to a key provider configuration. Its value can only be `openai`.
> | `keyIds` |  optional  | `[]string`         | Unique identifiers for keys.

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

```
KeyConfiguration
```
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | name | `string` | spike's developer key | Name of the API key. |
> | createdAt | `int64` | `1257894000` | Key configuration creation time in unix.  |
> | updatedAt | `int64` | `1257894000` | Key configuration update time in unix.  |
> | revoked | `boolean` | true | Indicator for whether the key is revoked.  |
> | revokedReason | `string` | The key has expired | Reason for why the key is revoked.  |
> | tags | `[]string` | ["org-tag-12345"]             | Identifiers associated with the key. |
> | keyId | `string` | `550e8400-e29b-41d4-a716-446655440000` | Unique identifier for the key.  |
> | costLimitInUsd | `float64` | `5.5` | Total spend limit of the API key.
> | costLimitInUsdOverTime | `float64` | `2` | Total spend within period of time. This field is required if costLimitInUsdUnit is specified.   |
> | costLimitInUsdUnit | `enum` | `d`                       | Time unit for costLimitInUsdOverTime. Possible values are [`m`, `h`, `d`, `mo`].      |
> | rateLimitOverTime | `int` | `2` | rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitUnit | `string` | `m`                         |  Time unit for rateLimitOverTime. Possible values are [`h`, `m`, `s`, `d`]       |
> | ttl | `string` | `24h` | time to live. Available units are [`s`, `m`, `h`] |
> | allowedPaths | `[]PathConfig` | `[{ "path": "/api/providers/openai/v1/chat/completion", "method": "POST"}]` | Allowed paths that can be accessed using the key. |
> | settingId | `string` | `98daa3ae-961d-4253-bf6a-322a32fdca3d` | This field is DEPERCATED. Use `settingIds` field instead.  |
> | settingIds | `string` | `[98daa3ae-961d-4253-bf6a-322a32fdca3d]` | Setting ids associated with the key. |
> | shouldLogRequest | `bool` | `false` | Should request be stored. |
> | shouldLogResponse | `bool` | `true` | Should response be stored. |
> | rotationEnabled | `bool` | `false` | Should key rotate setting used to access third party endpoints in order to circumvent rate limits. |
> | policyId | `string` | `98daa3ae-961d-4253-bf6a-322a32fdca3d` | Policy id associated with the key. |
> | isKeyNotHashed | `bool` | `false` | Flag controls whether or not the key should be hashed. |

</details>

<details>
  <summary>Get keys V2: <code>POST</code> <code><b>/api/v2/key-management/keys</b></code></summary>

##### Description
This endpoint is set up for retrieving key configurations using a query param called tag.

##### Request
> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | keyIds | optional | `[]string` | `["98daa3ae-961d-4253-bf6a-322a32fdca3d"]` | Array of key IDs. |
> | tags | optional | `[]string` | `["org-tag-12345"]`            | Identifiers associated with keys. |
> | name | optional | `string` | `key` | Filter keys by the substring match on the `name` field. |
> | revoked | optional | `bool` | `false` | Filters keys by `revoked` status.  |
> | limit | optional | `int` | `5` | Pagination limit. |
> | offset | optional | `int` | `5` | Pagination offset.
> | order | optional | `string` | `asc` | `asc` and `desc` are the two available values.   |
> | returnCount | optional | `bool` | `true`                       | Flag that controls whether the response should contain total key counts.      |

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
> | instance         | `string` | /api/v2/key-management/keys            |

##### Response
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | keys | `[]key` | Array of keys. |
> | count | `int` | 123 | Total number of keys.  |

###### key
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
> | costLimitInUsdUnit | `enum` | `d`                       | Time unit for costLimitInUsdOverTime. Possible values are [`m`, `h`, `d`, `mo`].      |
> | rateLimitOverTime | `int` | `2` | rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitUnit | `string` | `m`                         |  Time unit for rateLimitOverTime. Possible values are [`h`, `m`, `s`, `d`]       |
> | ttl | `string` | `24h` | time to live. Available units are [`s`, `m`, `h`] |
> | allowedPaths | `[]PathConfig` | `[{ "path": "/api/providers/openai/v1/chat/completion", method: "POST"}]` | Allowed paths that can be accessed using the key. |
> | settingId | `string` | `98daa3ae-961d-4253-bf6a-322a32fdca3d` | This field is DEPERCATED. Use `settingIds` field instead.  |
> | settingIds | `string` | `[98daa3ae-961d-4253-bf6a-322a32fdca3d]` | Setting ids associated with the key. |
> | shouldLogRequest | `bool` | `false` | Should request be stored. |
> | shouldLogResponse | `bool` | `true` | Should response be stored. |
> | rotationEnabled | `bool` | `false` | Should key rotate setting used to access third party endpoints in order to circumvent rate limits. |
> | policyId | `string` | `98daa3ae-961d-4253-bf6a-322a32fdca3d` | Policy id associated with the key. |
> | isKeyNotHashed | `bool` | `false` | Flag controls whether or not the key should be hashed. |
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
> | key | required | `string` | `abcdef12345` | API key. |
> | settingId | depercated | `string` | `98daa3ae-961d-4253-bf6a-322a32fdca3d` | This field is DEPERCATED. Use `settingIds` field instead.  |
> | settingIds | required | `string` | `98daa3ae-961d-4253-bf6a-322a32fdca3d` | Setting ids associated with the key. |
> | costLimitInUsd | optional | `float64` | `5.5` | Total spend limit of the API key.
> | costLimitInUsdOverTime | optional | `float64` | `2` | Total spend within period of time. This field is required if `costLimitInUsdUnit` is specified.   |
> | costLimitInUsdUnit | optional | `enum` | d                       | Time unit for costLimitInUsdOverTime. Possible values are [`m`, `h`, `d`, `mo`].      |
> | rateLimitOverTime | optional | `int` | `2` | rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitUnit | optional | `enum` | m                         |  Time unit for rateLimitOverTime. Possible values are [`h`, `m`, `s`, `d`]       |
> | ttl | optional | `string` | `24h` | time to live. Available units are [`s`, `m`, `h`]. |
> | allowedPaths | optional | `[]PathConfig` | `[{ "path": "/api/providers/openai/v1/chat/completion", method: "POST"}]` | Paths allowed for access. |
> | shouldLogRequest | optional | `bool` | `false` | Should request be stored. |
> | shouldLogResponse | optional | `bool` | `true` | Should response be stored. |
> | rotationEnabled | optional | `bool` | `false` | Should key rotate setting used to access third party endpoints in order to circumvent rate limits. |
> | policyId | optional | `string` | `98daa3ae-961d-4253-bf6a-322a32fdca3d` | Policy id associated with the key. |
> | isKeyNotHashed | optional | `bool` | `false` | Flag controls whether or not the key should be hashed. |



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
> | costLimitInUsdUnit | `enum` | d                       | Time unit for costLimitInUsdOverTime. Possible values are [`m`, `h`, `d`, `mo`].      |
> | rateLimitOverTime | `int` | `2` | rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitOverTime | `int` | `2` | rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitUnit | `string` | m                         |  Time unit for rateLimitOverTime. Possible values are [`h`, `m`, `s`, `d`].       |
> | ttl | `string` | 2d | time to live. Available units are [`s`, `m`, `h`] |
> | allowedPaths | `[]PathConfig` | `[{ "path": "/api/providers/openai/v1/chat/completion", method: "POST"}]` | Allowed paths that can be accessed using the key. |
> | settingId | `string` | `98daa3ae-961d-4253-bf6a-322a32fdca3d` | This field is DEPERCATED. Use `settingIds` field instead.  |
> | settingIds | `string` | `[98daa3ae-961d-4253-bf6a-322a32fdca3d]` | Setting ids associated with the key. |
> | shouldLogRequest | `bool` | `false` | Should request be stored. |
> | shouldLogResponse | `bool` | `true` | Should response be stored. |
> | rotationEnabled | `bool` | `false` | Should key rotate setting used to access third party endpoints in order to circumvent rate limits. |
> | policyId | `string` | `98daa3ae-961d-4253-bf6a-322a32fdca3d` | Policy id associated with the key. |
> | isKeyNotHashed | `bool` | `false` | Flag controls whether or not the key should be hashed. |

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
> | settingId | optional | `string` | 98daa3ae-961d-4253-bf6a-322a32fdca3d | This field is DEPERCATED. Use `settingIds` field instead.  |
> | settingIds | optional | `string` | 98daa3ae-961d-4253-bf6a-322a32fdca3d | Setting ids associated with the key. |
> | name | optional | `string` | spike's developer key | Name of the API key. |
> | tags | optional | `[]string` | `["org-tag-12345"]`             | Identifiers associated with the key. |
> | revoked | optional |  `boolean` | `true` | Indicator for whether the key is revoked.  |
> | revokedReason | optional | `string` | The key has expired | Reason for why the key is revoked.  |
> | costLimitInUsd | optional | `float64` | `5.5` | Total spend limit of the API key.
> | costLimitInUsdOverTime | optional | `float64` | `2` | Total spend within period of time. This field is required if costLimitInUsdUnit is specified.   |
> | costLimitInUsdUnit | optional | `enum` | `d`                       | Time unit for costLimitInUsdOverTime. Possible values are [`m`, `h`, `d`, `mo`].      |
> | rateLimitOverTime | optional | `int` | `2` | rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitUnit | optional | `string` | `m`                         |  Time unit for rateLimitOverTime. Possible values are [`h`, `m`, `s`, `d`]       |
> | allowedPaths | optional | `[{ "path": "/api/providers/openai/v1/chat/completions", "method": "POST"}]` | `` | Paths allowed for access. |
> | shouldLogRequest | optional | `bool` | `false` | Should request be stored. |
> | shouldLogResponse | optional | `bool` | `true` | Should response be stored. |
> | rotationEnabled | optional | `bool` | `false` | Should key rotate setting used to access third party endpoints in order to circumvent rate limits. |
> | policyId | optional | `string` | `98daa3ae-961d-4253-bf6a-322a32fdca3d` | Policy id associated with the key. |
> | isKeyNotHashed | optional | `bool` | `false` | Flag controls whether or not the key should be hashed. |

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
> | costLimitInUsdUnit | `enum` | `d`                       | Time unit for costLimitInUsdOverTime. Possible values are [`m`, `h`, `d`, `mo`].      |
> | rateLimitOverTime | `int` | `2` | rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitUnit | `string` | `m`                         |  Time unit for rateLimitOverTime. Possible values are [`h`, `m`, `s`, `d`]       |
> | ttl | `string` | `2d` | time to live. Available units are [`s`, `m`, `h`] |
> | allowedPaths | `[]PathConfig` | `[{ "path": "/api/providers/openai/v1/chat/completion", method: "POST"}]` | Allowed paths that can be accessed using the key. |
> | settingId | `string` | `98daa3ae-961d-4253-bf6a-322a32fdca3d` | This field is DEPERCATED. Use `settingIds` field instead.  |
> | settingIds | `string` | `[98daa3ae-961d-4253-bf6a-322a32fdca3d]` | Setting ids associated with the key. |
> | shouldLogRequest | `bool` | `false` | Should request be stored. |
> | shouldLogResponse | `bool` | `true` | Should response be stored. |
> | rotationEnabled | `bool` | `false` | Should key rotate setting used to access third party endpoints in order to circumvent rate limits. |
> | policyId | `string` | `98daa3ae-961d-4253-bf6a-322a32fdca3d` | Policy id associated with the key. |
> | isKeyNotHashed | `bool` | `false` | Flag controls whether or not the key should be hashed. |

</details>

<details>
  <summary>Get top spending key Ids: <code>POST</code> <code><b>/api/reporting/top-keys</b></code></summary>

##### Description
This endpoint is set up for getting a list of key ids ordered by spend.

##### Request
> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | start | required | `int64` | `1257894000` | Start unix timestamp. |
> | end | required | `int64` | `1257894000` | End unix timestamp  |
> | keyIds | optional | `[]string` | `["98daa3ae-961d-4253-bf6a-322a32fdca3d"]` | Array of key IDs. |
> | tags | optional | `[]string` | `["org-tag-12345"]`            | Identifiers associated with keys. |
> | name | optional | `string` | `key` | Filter keys by the substring match on the `name` field. |
> | revoked | optional | `bool` | `false` | Filters keys by `revoked` status.  |
> | limit | optional | `int` | `5` | Pagination limit. |
> | offset | optional | `int` | `5` | Pagination offset.
> | order | optional | `string` | `asc` | `asc` and `desc` are the two available values.   |

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
> | instance         | `string` | /api/reporting/top-keys            |

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
> | costLimitInUsdUnit | `enum` | `d`                       | Time unit for costLimitInUsdOverTime. Possible values are [`m`, `h`, `d`, `mo`].      |
> | rateLimitOverTime | `int` | `2` | rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitUnit | `string` | `m`                         |  Time unit for rateLimitOverTime. Possible values are [`h`, `m`, `s`, `d`]       |
> | ttl | `string` | `2d` | time to live. Available units are [`s`, `m`, `h`] |
> | allowedPaths | `[]PathConfig` | `[{ "path": "/api/providers/openai/v1/chat/completion", method: "POST"}]` | Allowed paths that can be accessed using the key. |
> | settingId | `string` | `98daa3ae-961d-4253-bf6a-322a32fdca3d` | This field is DEPERCATED. Use `settingIds` field instead.  |
> | settingIds | `string` | `[98daa3ae-961d-4253-bf6a-322a32fdca3d]` | Setting ids associated with the key. |
> | shouldLogRequest | `bool` | `false` | Should request be stored. |
> | shouldLogResponse | `bool` | `true` | Should response be stored. |
> | rotationEnabled | `bool` | `false` | Should key rotate setting used to access third party endpoints in order to circumvent rate limits. |
> | policyId | `string` | `98daa3ae-961d-4253-bf6a-322a32fdca3d` | Policy id associated with the key. |
> | isKeyNotHashed | `bool` | `false` | Flag controls whether or not the key should be hashed. |

</details>

<details>
  <summary>Create a provider setting: <code>POST</code> <code><b>/api/provider-settings</b></code></summary>

##### Description
This endpoint is creating a provider setting.

##### Request
> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | provider | required | `enum` | openai | This value can only be `openai`, `anthropic` and `azure` as for now. |
> | setting | required | `Setting` | `{ "apikey": "YOUR_OPENAI_KEY" }`            | A map of values used for authenticating with the selected provider. |
> | name | optional | `string` | YOUR_PROVIDER_SETTING_NAME | This field is used for giving a name to provider setting |
> | allowedModels | `[]string` | `["text-embedding-ada-002"]` | Allowed models for this provider setting. |

```Setting```
> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | apiKey | required | `string` | `xx-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`  | This value is required. |
> | url | required | `string` | `https://your.deployment.url`  | This value is required when the provider is `vllm` |
> | resourceName | required | `string` | `YOUR_AZURE_RESOURCE_NAME`            | This value is required when the provider is `azure`. |


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
> | id | `string` | `98daa3ae-961d-4253-bf6a-322a32fdca3d` | This value is a unique identifier. |
> | name | `string` | `YOUR_PROVIDER_SETTING_NAME` | Provider setting name. |
> | allowedModels | `[]string` | `["text-embedding-ada-002"]` | Allowed models for this provider setting. |
</details>


<details>
  <summary>Get provider settings: <code>GET</code> <code><b>/api/provider-settings</b></code></summary>

##### Description
This endpoint is getting provider settings.

##### Query Parameters
> | name   |  type      | data type      | description                                          |
> |--------|------------|----------------|------------------------------------------------------|
> | `ids` |  optional   | `[]string`         | Provider setting ids                 |

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
> | id | `string` | `98daa3ae-961d-4253-bf6a-322a32fdca3d` | This value is a unique identifier. |
> | name | `string` | `YOUR_PROVIDER_SETTING_NAME` | Provider setting name. |
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
> | setting | required | `Setting` | `{ "apikey": "YOUR_OPENAI_KEY" }`            | A map of values used for authenticating with the selected provider. |
> | name | optional | `string` | `YOUR_PROVIDER_SETTING_NAME` | This field is used for giving a name to provider setting |
> | allowedModels | `[]string` | `["text-embedding-ada-002"]` | Allowed models for this provider setting. |

```Setting```
> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | apiKey | required | `string` | `xx-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`  | This value is required. |
> | url | required | `string` | `https://your.deployment.url`  | This value is required when the provider is `vllm` |
> | resourceName | required | `string` | `YOUR_AZURE_RESOURCE_NAME`            | This value is required when the provider is `azure`. |


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
> | name | `string` | `YOUR_PROVIDER_SETTING_NAME` | Provider setting name. |
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
> | tags | required | `[]string` | `["tag-1", "tag-2"]`           | Array of tags that specicify the key tags that you want to aggregate stats from. |
> | customIds | required | `[]string` | `["customId-1", "customId-2"]` | A list of custom IDs that you want to aggregate stats from. |
> | filters | required | `[]string` | `["model", "keyId"]` | Group by data points through different filters(`model`,`keyId`, `userId` or `customId`). |
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

Datapoint
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | timeStamp | `int64` | `1702504746` | Unix timestamp for the data point |
> | numberOfRequests | `int64` | `100` | Aggregated number of http requests over the given time increment. |
> | costInUsd | `float64` | `1.7` | Aggregated cost of proxied requests in USD over the given time increment. |
> | latencyInMs | `int` | `555` | Aggregated latency in milliseconds of http requests over the given time increment. |
> | promptTokenCount | `int` | `25` | Aggregated prompt token counts over the given time increment. |
> | completionTokenCount | `int` | `4000` | Aggregated completion token counts over the given time increment. |
> | successCount | `int` | `555` | Aggregated number of successful http requests over the given time increment. |
> | keyId | `int` | `555.7` | key Id associated with the event. |
> | model | `string` | `gpt-3.5-turbo` | model associated with the event. |
> | customId | `string` | `customId` | customId associated with the event. |

</details>

<details>
  <summary>Get events: <code>GET</code> <code><b>/api/events</b></code></summary>

##### Description
This endpoint is for getting events.

##### Query Parameters
> | name   |  type      | data type      | description                                          |
> |--------|------------|----------------|------------------------------------------------------|
> | `customId` |  optional   | `string`         | Custom identifier attached to an event.                  |
> | `keyIds` |  optional   | `[]string`         | A list of key IDs.                 |
> | `start` |  required if `keyIds` is specified   | `int64`         | Start timestamp.                |
> | `end` |  required if `keyIds` is specified   | `int64`         | End timestamp.                |

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
> | key_id | `string` | `YOUR_KEY_ID` | Key Id associated with the proxy request. |
> | cost_in_usd | `float64` | `0.0004` | Cost incured by the proxy request. |
> | model | `string` | `gpt-4-1105-preview` | Model used in the proxy request. |
> | provider | `string` | `openai` | Provider for the proxy request. |
> | status | `int` | `200` | Http status. |
> | prompt_token_count | `int` | `8` | Prompt token count of the proxy request. |
> | completion_token_count | `int` | `16` | Completion token counts of the proxy request. |
> | latency_in_ms | `int` | `160` | Provider setting name. |
> | path | `string` | `/api/v1/chat/completion` | Provider setting name. |
> | method | `string` | `POST` | Http method for the assoicated proxu request. |
> | custom_id | `string` | `YOUR_CUSTOM_ID` | Custom Id passed by the user in the headers of proxy requests. |
> | request | `[]byte` | `{}` | Request in bytes to providers. |
> | custom_id | `string` | `YOUR_CUSTOM_ID` | Custom Id passed by the user in the headers of proxy requests. |
> | response | `[]byte` | `{}` | Response in bytes from providers. |

</details>

<details>
  <summary>Get events V2: <code>GET</code> <code><b>/api/v2/events</b></code></summary>

##### Description
This endpoint retrieves a list of events based on provided filters.

##### Request
> | name              | type       | data type       | example               | description                                                     |
> |-------------------|------------|-----------------|-----------------------|-----------------------------------------------------------------|
> | `userIds`         | optional   | `[]string`      | `["user1", "user2"]`  | List of user IDs to filter events by specific users.            |
> | `customIds`       | optional   | `[]string`      | `["id1", "id2"]`      | List of custom identifiers for filtering events.                |
> | `keyIds`          | optional   | `[]string`      | `["key1", "key2"]`    | List of key IDs used to filter events.                          |
> | `tags`            | optional   | `[]string`      | `["tag1", "tag2"]`    | List of tags associated with events for filtering.              |
> | `start`           | required   | `int64`         | `1609459200`          | Start timestamp for filtering events.                           |
> | `end`             | required   | `int64`         | `1612137600`          | End timestamp for filtering events.                             |
> | `limit`           | optional   | `int`           | `100`                 | Maximum number of events to return.                             |
> | `offset`          | optional   | `int`           | `0`                   | Offset for pagination.                                          |
> | `policyIds`       | optional   | `[]string`      | `["policy1", "policy2"]` | List of policy IDs to filter events related to specific policies.|
> | `actions`         | optional   | `[]string`      | `["allowed"]` | Values can include `allowed`, `redacted`, and `blocked`.                            |
> | `costOrder`       | optional   | `string`        | `asc`               | Order by cost, specify `asc` or `desc`.                         |
> | `dateOrder`       | optional   | `string`        | `desc`              | Order by date, specify 'asc' or 'desc'.                         |
> | `returnCount`     | optional   | `bool`          | `true`                | Whether to return the count of events instead of event details. |

##### Response
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | events | `[]Event` | `[]` | List of events.  |
> | count | `int` | `20` | Total number of events returned.  |

Event
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | id | `int64` | `1699933571` | Unique identifier associated with the event.  |
> | created_at | `int64` | `1699933571` | Unix timestamp for creation time.  |
> | tags | `int64` | `["YOUR_TAG"]` | Tags of the key. |
> | key_id | `string` | `YOUR_KEY_ID` | Key Id associated with the proxy request. |
> | cost_in_usd | `float64` | `0.0004` | Cost incured by the proxy request. |
> | model | `string` | `gpt-4-1105-preview` | Model used in the proxy request. |
> | provider | `string` | `openai` | Provider for the proxy request. |
> | status | `int` | `200` | Http status. |
> | prompt_token_count | `int` | `8` | Prompt token count of the proxy request. |
> | completion_token_count | `int` | `16` | Completion token counts of the proxy request. |
> | latency_in_ms | `int` | `160` | Provider setting name. |
> | path | `string` | `/api/v1/chat/completion` | Provider setting name. |
> | method | `string` | `POST` | Http method for the assoicated proxu request. |
> | custom_id | `string` | `YOUR_CUSTOM_ID` | Custom Id passed by the user in the headers of proxy requests. |
> | request | `[]byte` | `{}` | Request in bytes to providers. |
> | custom_id | `string` | `YOUR_CUSTOM_ID` | Custom Id passed by the user in the headers of proxy requests. |
> | response | `[]byte` | `{}` | Response in bytes from providers. |

</details>

<details>
  <summary>Create custom provider: <code>POST</code> <code><b>/api/custom/providers</b></code></summary>

##### Description
This endpoint is creating custom providers.

##### RouteConfig
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

##### RouteConfig
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
  <summary>Create a policy: <code>POST</code> <code><b>/api/policies</b></code></summary>

##### Description
This endpoint is for creating a policy.

##### Request
> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | name | optional | `string` | `Pivacy Policy  #1` | Name for the policy. |
> | tags | required | `[]string` | `["orgId-1"]` | Associated tags. |
> | config | optional | `config` | `{"rules": { "address": "block" }}` | PII detection rules. |
> | regexConfig | optional | `regexConfig` | `{"rules": [{"definition": "[2-9]\|[12]\d\|3[0-6]", "action": "block"}]}` | Regular expression rules. |

##### Config
> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | rules | required | `map[Rule]Action` | `{ "address": "block" }` | PII entities mapped to their associated actions. |

##### RegexConfig
> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | definition | required | `string` | `[2-9]\|[12]\d\|3[0-6]` | Regex definition. |
> | action | required | `Action` | `block` | Action associated with the regex definition. |

##### Rule
> | type | example | description |
> | `enum`| `address` | Possible values are `address`,`age`,`all`,`aws_access_key`,`aws_secret_key`,`bank_account_number`,`bank_routing`,`ca_health_number`,`ca_social_insurance_number`,`credit_debit_cvv`,`credit_debit_expiry`,`credit_debit_number`,`date_time`,`driver_id`,`email`,`in_aadhaar`,`in_nrega`,`in_permanent_account_number`,`in_voter_number`,`international_bank_account_number`,`ip_address`,`license_plate`,`mac_address`,`name`,`passport_number`,`password`,`phone`,`pin`,`ssn`,`swift_code`,`uk_national_health_service_number`,`uk_national_insurance_number`,`uk_unique_taxpayer_reference_number`,`url`,`us_individual_tax_identification_number`,`username`, and `vehicle_identification_number`. |

##### Action
> | type | example | description |
> | `enum`| `block` | Possible values are `block`,`allow_but_redact`, and `allow`. |

##### Error Response
> | http code     | content-type                      |
> |---------------|-----------------------------------|
> | `500`         | `application/json`                |

> | Field     | type | example                      |
> |---------------|-----------------------------------|-|
> | status         | `int` | `400`, `500`           |
> | title         | `string` | `request body reader error `            |
> | type         | `string` | `/errors/policies`             |
> | detail         | `string` | `something is wrong`            |
> | instance         | `string` | `/api/policies`  |

##### Response
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | id | `int64` | `9e6e8b27-2ce0-4ef0-bdd7-1ed3916592eb` | Unique identifier associated with the event.  |
> | created_at | `int64` | `1699933571` | Unix timestamp for creation time.  |
> | updated_at | `int64` | `1699933571` | Unix timestamp for update time.  |
> | tags | `[]string` | `["org-111"]`  | Tags attached to policies. |
> | config | `config` | `{"rules": { "address": "block" }}` | PII detection rules. |
> | regexConfig | `regexConfig` | `{"rules": [{"definition": "[2-9]\|[12]\d\|3[0-6]", "action": "block"}]}` | Regular expression rules. |
</details>


<details>
  <summary>Update a policy: <code>PATCH</code> <code><b>/api/policies/:id</b></code></summary>

##### Description
This endpoint is for updating a policy.

##### Request
> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | name | optional | `string` | `Pivacy Policy  #1` | Name for the policy. |
> | tags | required | `[]string` | `["orgId-1"]` | Associated tags. |
> | config | optional | `config` | `{"rules": { "address": "block" }}` | PII detection rules. |
> | regexConfig | optional | `regexConfig` | `{"rules": [{"definition": "[2-9]\|[12]\d\|3[0-6]", "action": "block"}]}` | Regular expression rules. |

##### Config
> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | rules | required | `map[Rule]Action` | `{ "address": "block" }` | PII entities mapped to their associated actions. |

##### RegexConfig
> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | definition | required | `string` | `[2-9]\|[12]\d\|3[0-6]` | Regex definition. |
> | action | required | `Action` | `block` | Action associated with the regex definition. |

##### Rule
> | type | example | description |
> | `enum`| `address` | Possible values are `address`,`age`,`all`,`aws_access_key`,`aws_secret_key`,`bank_account_number`,`bank_routing`,`ca_health_number`,`ca_social_insurance_number`,`credit_debit_cvv`,`credit_debit_expiry`,`credit_debit_number`,`date_time`,`driver_id`,`email`,`in_aadhaar`,`in_nrega`,`in_permanent_account_number`,`in_voter_number`,`international_bank_account_number`,`ip_address`,`license_plate`,`mac_address`,`name`,`passport_number`,`password`,`phone`,`pin`,`ssn`,`swift_code`,`uk_national_health_service_number`,`uk_national_insurance_number`,`uk_unique_taxpayer_reference_number`,`url`,`us_individual_tax_identification_number`,`username`, and `vehicle_identification_number`. |

##### Action
> | type | example | description |
> | `enum`| `block` | Possible values are `block`,`allow_but_redact`, and `allow`. |

##### Error Response
> | http code     | content-type                      |
> |---------------|-----------------------------------|
> | `500`         | `application/json`                |

> | Field     | type | example                      |
> |---------------|-----------------------------------|-|
> | status         | `int` | `400`, `500`           |
> | title         | `string` | `request body reader error `            |
> | type         | `string` | `/errors/policies`             |
> | detail         | `string` | `something is wrong`            |
> | instance         | `string` | `/api/policies`  |

##### Response
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | id | `int64` | `9e6e8b27-2ce0-4ef0-bdd7-1ed3916592eb` | Unique identifier associated with the event.  |
> | created_at | `int64` | `1699933571` | Unix timestamp for creation time.  |
> | updated_at | `int64` | `1699933571` | Unix timestamp for update time.  |
> | tags | `[]string` | `["org-111"]`  | Tags attached to policies. |
> | config | `config` | `{"rules": { "address": "block" }}` | PII detection rules. |
> | regexConfig | `regexConfig` | `{"rules": [{"definition": "[2-9]\|[12]\d\|3[0-6]", "action": "block"}]}` | Regular expression rules. |
</details>


<details>
  <summary>Get policies by tags: <code>GET</code> <code><b>/api/policies</b></code></summary>

##### Description
This endpoint is for retrieving policies by tags.

##### Query Parameters
> | name   |  type      | data type      | description                                          |
> |--------|------------|----------------|------------------------------------------------------|s
> | `tags` |  required  | `[]string`         | Tags attached to the policies.                  |

##### Response
```
[]Policy
```
Policy
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | id | `int64` | `9e6e8b27-2ce0-4ef0-bdd7-1ed3916592eb` | Unique identifier associated with the event.  |
> | created_at | `int64` | `1699933571` | Unix timestamp for creation time.  |
> | updated_at | `int64` | `1699933571` | Unix timestamp for update time.  |
> | tags | `[]string` | `["org-111"]`  | Tags attached to policies. |
> | config | `config` | `{"rules": { "address": "block" }}` | PII detection rules. |
> | regexConfig | `regexConfig` | `{"rules": [{"definition": "[2-9]\|[12]\d\|3[0-6]", "action": "block"}]}` | Regular expression rules. |

</details>

<details>
  <summary>Get custom providers: <code>GET</code> <code><b>/api/custom/providers</b></code></summary>

##### Description
This endpoint is for getting custom providers.

##### RouteConfig
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
> | id | `int64` | `9e6e8b27-2ce0-4ef0-bdd7-1ed3916592eb` | Unique identifier associated with the event.  |
> | created_at | `int64` | `1699933571` | Unix timestamp for creation time.  |
> | updated_at | `int64` | `1699933571` | Unix timestamp for update time.  |
> | provider | `string` | `bricks`  | Unique identifier associated with the route config. |
> | route_configs | `[]RouteConfig` | `{{ "path": "/chat/completions", "target_url": "https://api.openai.com/v1/chat/completions" }}` | Start timestamp for the requested timeseries data. |
> | authentication_param | `string` | `apikey` | The authentication parameter required for. |
</details>

<details>
  <summary>Create routes: <code>POST</code> <code><b>/api/routes</b></code></summary>

##### Description
This endpoint is for creating routes.

##### StepConfig
> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | provider | required | `enum` | `azure` | Provider for the step. Can only be either `azure` or `openai`. |
> | model | required | `string` | `gpt-3.5-turbo` | Model that the step should call. Can only be chat completion or embedding models from OpenAI or Azure OpenAI. |
> | retries | optional | `int` | `2` | Number of retries. |
> | params | optional | `object` | `{ deploymentId: "ada-test",apiVersion: "2022-12-01" }` | Params required for maing API requests to desired modela and provider combo. Required if the provider is `azure` |
> | timeout | optional | `string` | `5s` | Timeout desired for each request. Default value is `5m`. |


##### CacheConfig
> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | enabled | required | `bool` | `false` | Boolean flag indicating whether caching is enabled. |
> | ttl | optional | `string` | `5s` | TTL for the cache. Default value is `168h`. |

##### Request
> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | name | required | `string` | `staging-openai-azure-completion-route` | Name for the route. |
> | path | required | `string` | `/` | Unique identifier for. |
> | steps | required | `[]StepConfig` | `apikey` | The authentication parameter required for. |
> | keyIds | required | `[]string` | `[]` | The authentication parameter required for. |
> | cacheConfig | required | `CacheConfig` | `[]` | The authentication parameter required for. |

##### Error Response
> | http code     | content-type                      |
> |---------------|-----------------------------------|
> | `500, 400`        | `application/json`                |

> | Field     | type | example                      |
> |---------------|-----------------------------------|-|
> | status         | `int` | `400`            |
> | title         | `string` | `request body reader error`             |
> | type         | `string` | `/errors/request-body-read`             |
> | detail         | `string` | `something is wrong`            |
> | instance         | `string` | `/api/routes`           |

##### Response
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | id | required | `string` | `9e6e8b27-2ce0-4ef0-bdd7-1ed3916592eb` | Unique identifier for route. |
> | createdAt | required | `string` | `1699933571` | Creation time of the route. |
> | updatedAt | required | `string` | `1699933571` | Update time of the route. |
> | name | required | `string` | `staging-openai-azure-completion-route` | Name for the route. |
> | path | required | `string` | `/production/chat/completion` | Unique path for the route. |
> | steps | required | `[]StepConfig` | `[{"retries": 2, "provider": "openai", "params": {}, "model": "gpt-3.5-turbo", "timeout": "1s"}]` | List of steps configurations that details sequences of API calls. |
> | keyIds | required | `[]string` | `["9e6e8b27-2ce0-4ef0-bdd7-1ed3916592eb"]` | List of key IDs that can be used to access the route. |
> | cacheConfig | required | `CacheConfig` | `{ "enabled": false, "ttl": "5s" }` | The caching configurations parameter required for. |
</details>

<details>
  <summary>Retrieve a route: <code>GET</code> <code><b>/api/routes/:id</b></code></summary>

##### Description
This endpoint is for retrieving a route.

##### Parameters
> | name   |  type      | data type      | description                                          |
> |--------|------------|----------------|------------------------------------------------------|
> | `id` |  required  | `string`         | Unique identifier for the route.     

##### Error Response
> | http code     | content-type                      |
> |---------------|-----------------------------------|
> | `500, 404`        | `application/json`                |

> | Field     | type | example                      |
> |---------------|-----------------------------------|-|
> | status         | `int` | `404`            |
> | title         | `string` | `request body reader error`             |
> | type         | `string` | `/errors/request-body-read`             |
> | detail         | `string` | `something is wrong`            |
> | instance         | `string` | `/api/routes/:id`           |

##### Response
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | id | required | `string` | `9e6e8b27-2ce0-4ef0-bdd7-1ed3916592eb` | Unique identifier for route. |
> | createdAt | required | `string` | `1699933571` | Creation time of the route. |
> | updatedAt | required | `string` | `1699933571` | Update time of the route. |
> | name | required | `string` | `staging-openai-azure-completion-route` | Name for the route. |
> | path | required | `string` | `/production/chat/completion` | Unique path for the route. |
> | steps | required | `[]StepConfig` | `[{"retries": 2, "provider": "openai", "params": {}, "model": "gpt-3.5-turbo", "timeout": "1s"}]` | List of steps configurations that details sequences of API calls. |
> | keyIds | required | `[]string` | `["9e6e8b27-2ce0-4ef0-bdd7-1ed3916592eb"]` | List of key IDs that can be used to access the route. |
> | cacheConfig | required | `CacheConfig` | `{ "enabled": false, "ttl": "5s" }` | The caching configurations parameter required for. |
</details>

<details>
  <summary>Retrieve routes: <code>GET</code> <code><b>/api/routes</b></code></summary>

##### Description
This endpoint is for retrieving routes.

##### Error Response
> | http code     | content-type                      |
> |---------------|-----------------------------------|
> | `500`        | `application/json`                |

> | Field     | type | example                      |
> |---------------|-----------------------------------|-|
> | status         | `int` | `404`            |
> | title         | `string` | `request body reader error`             |
> | type         | `string` | `/errors/request-body-read`             |
> | detail         | `string` | `something is wrong`            |
> | instance         | `string` | `/api/routes/:id`           |

##### CacheConfig
> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | enabled | required | `bool` | `false` | Boolean flag indicating whether caching is enabled. |
> | ttl | optional | `string` | `5s` | TTL for the cache. Default value is `168h`. |

##### StepConfig
> | Field | required | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | provider | required | `enum` | `azure` | Provider for the step. Can only be either `azure` or `openai`. |
> | model | required | `string` | `gpt-3.5-turbo` | Model that the step should call. Can only be chat completion or embedding models from OpenAI or Azure OpenAI. |
> | retries | optional | `int` | `2` | Number of retries. |
> | params | optional | `object` | `{ deploymentId: "ada-test",apiVersion: "2022-12-01" }` | Params required for maing API requests to desired modela and provider combo. Required if the provider is `azure` |
> | timeout | optional | `string` | `5s` | Timeout desired for each request. Default value is `5m`. |


##### RouteConfig
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | id | required | `string` | `9e6e8b27-2ce0-4ef0-bdd7-1ed3916592eb` | Unique identifier for route. |
> | createdAt | required | `string` | `1699933571` | Creation time of the route. |
> | updatedAt | required | `string` | `1699933571` | Update time of the route. |
> | name | required | `string` | `staging-openai-azure-completion-route` | Name for the route. |
> | path | required | `string` | `/production/chat/completion` | Unique path for the route. |
> | steps | required | `[]StepConfig` | `[{"retries": 2, "provider": "openai", "params": {}, "model": "gpt-3.5-turbo", "timeout": "1s"}]` | List of steps configurations that details sequences of API calls. |
> | keyIds | required | `[]string` | `["9e6e8b27-2ce0-4ef0-bdd7-1ed3916592eb"]` | List of key IDs that can be used to access the route. |
> | cacheConfig | required | `CacheConfig` | `{ "enabled": false, "ttl": "5s" }` | The caching configurations parameter required for. |


##### Response
```
[]RouteConfig
```
</details>

<details>
  <summary>Get user ids: <code>GET</code> <code><b>/api/reporting/users-ids</b></code></summary>

##### Description
This endpoint is for getting user ids given a key id.

##### Query Parameters
> | name   |  type      | data type      | description                                          |
> |--------|------------|----------------|------------------------------------------------------|
> | `keyId` |  required   | `string`         | Key Id               |

##### Error Response
> | http code     | content-type                      |
> |---------------|-----------------------------------|
> | `500`        | `application/json`                |

> | Field     | type | example                      |
> |---------------|-----------------------------------|-|
> | status         | `int` | `500`            |
> | title         | `string` | request body reader error             |
> | type         | `string` | /errors/request-body-read             |
> | detail         | `string` | something is wrong            |
> | instance         | `string` | /api/reporting/user-ids           |

##### Response
```
[]string
```
</details>

<details>
  <summary>Get custom ids: <code>GET</code> <code><b>/api/reporting/custom-ids</b></code></summary>

##### Description
This endpoint is for getting custom ids given a key id.

##### Query Parameters
> | name   |  type      | data type      | description                                          |
> |--------|------------|----------------|------------------------------------------------------|
> | `keyId` |  required   | `string`         | Key Id               |

##### Error Response
> | http code     | content-type                      |
> |---------------|-----------------------------------|
> | `500`        | `application/json`                |

> | Field     | type | example                      |
> |---------------|-----------------------------------|-|
> | status         | `int` | `500`            |
> | title         | `string` | request body reader error             |
> | type         | `string` | /errors/request-body-read             |
> | detail         | `string` | something is wrong            |
> | instance         | `string` | /api/reporting/custom-ids           |

##### Response
```
[]string
```
</details>

<details>
  <summary>Create a user: <code>POST</code> <code><b>/api/users</b></code></summary>

##### Description
This endpoint is creating a user.

##### Query Parameters
> | name   |  type      | data type      | description                                          |
> |--------|------------|----------------|------------------------------------------------------|
> | `tag` |  optional   | `string`         | Identifier attached to a key configuration                  |
> | `tags` |  optional  | `[]string`         | Identifiers attached to a key configuration                  |
> | `provider` |  optional  | `string`         | Provider attached to a key provider configuration. Its value can only be `openai`.
> | `keyIds` |  optional  | `[]string`         | Unique identifiers for keys.

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
> | keyIds | optional | `[]string` | `["98daa3ae-961d-4253-bf6a-322a32fdca3d"]` | Setting ids associated with the key. |
> | costLimitInUsd | optional | `float64` | `5.5` | Total spend limit of the API key.
> | costLimitInUsdOverTime | optional | `float64` | `2` | Total spend within period of time. This field is required if `costLimitInUsdUnit` is specified.   |
> | costLimitInUsdUnit | optional | `enum` | d                       | Time unit for costLimitInUsdOverTime. Possible values are [`m`, `h`, `d`, `mo`].      |
> | rateLimitOverTime | optional | `int` | 2 | rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitUnit | optional | `enum` | m                         |  Time unit for rateLimitOverTime. Possible values are [`h`, `m`, `s`, `d`]       |
> | ttl | optional | `string` | 2d | time to live. Available units are [`s`, `m`, `h`]. |
> | allowedPaths | optional | `[]PathConfig` | `[{ "path": "/api/providers/openai/v1/chat/completion", "method": "POST"}]` | List of paths that can be accessed by the user. |
> | allowedModels | optional | `[]string` | `["gpt-4"]` | List of models that can be accessed by the user. |
> | userId | optional | `string` | `98daa3ae-961d-4253-bf6a-322a32fdca3d` | Client defined user ID. |

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
> | instance         | `string` | /api/users           |

##### Response
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | id | `string` | `550e8400-e29b-41d4-a716-446655440000` | Id of the user object. |
> | name | `string` | spike's developer key | Name of the user. |
> | createdAt | `int64` | 1257894000 | UserKey configuration creation time in unix.  |
> | updatedAt | `int64` | 1257894000 | User configuration update time in unix.  |
> | tags | `[]string` | ["org-tag-12345"]             | Identifiers associated with the user. |
> | keyIds | `[]string` | `["550e8400-e29b-41d4-a716-446655440000"]` | Array of key IDs.  |
> | revoked | `boolean` | true | Indicator for whether the user is revoked.  |
> | revokedReason | `string` | The key has expired | Reason for why the user is revoked.  |
> | costLimitInUsd | `float64` | `5.5` | Total spend limit of the user.
> | costLimitInUsdOverTime | `float64` | `2` | Total spend within period of time. This field is required if costLimitInUsdUnit is specified.   |
> | costLimitInUsdUnit | `enum` | d                       | Time unit for costLimitInUsdOverTime. Possible values are [`m`, `h`, `d`, `mo`].      |
> | rateLimitOverTime | `int` | `2` | rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitUnit | `string` | m                         |  Time unit for rateLimitOverTime. Possible values are [`h`, `m`, `s`, `d`]       |
> | ttl | `string` | `24h` | time to live. Available units are [`s`, `m`, `h`] |
> | allowedPaths | `[]PathConfig` | `[{ "path": "/api/providers/openai/v1/chat/completion", "method": "POST"}]` | List of paths that can be accessed by the user. |
> | allowedModels | `[]string` | `["gpt-4"]` | List of models that can be accessed by the user.  |
> | userId | `string` | `98daa3ae-961d-4253-bf6a-322a32fdca3d` | Client defined user id. |
</details>

<details>
  <summary>Update a user via id: <code>PATCH</code> <code><b>/api/users/:id</b></code></summary>

##### Description
This endpoint is updating a user via id.

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
> | name | required | `string` | `spike` | Name of the user. |
> | keyIds | optional | `[]string` | `["98daa3ae-961d-4253-bf6a-322a32fdca3d"]` | Keys ids associated with the user. |
> | costLimitInUsd | optional | `float64` | `5.5` | Total spend limit of the user.
> | revoked | optional | `boolean` | `true` | Indicator for whether the user is revoked.  |
> | revokedReason | optional | `string` | `expired` | Reason for why the user is revoked.  |
> | costLimitInUsdOverTime | optional | `float64` | `2` | Total spend within period of time. This field is required if `costLimitInUsdUnit` is specified.   |
> | costLimitInUsdUnit | optional | `enum` | d                       | Time unit for costLimitInUsdOverTime. Possible values are [`m`, `h`, `d`, `mo`].      |
> | rateLimitOverTime | optional | `int` | 2 | Rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitUnit | optional | `enum` | m                         |  Time unit for rateLimitOverTime. Possible values are [`h`, `m`, `s`, `d`]       |
> | ttl | optional | `string` | `24h` | time to live. Available units are [`s`, `m`, `h`]. |
> | allowedPaths | optional | `[]PathConfig` | `[{ "path": "/api/providers/openai/v1/chat/completion", "method": "POST"}]` | List of paths that can be accessed by the user. |
> | allowedModels | optional | `[]string` | `["gpt-4"]` | List of models that can be accessed by the user. |

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
> | instance         | `string` | /api/users           |

##### Response
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | id | `string` | `550e8400-e29b-41d4-a716-446655440000` | Id of the user object. |
> | name | `string` | `spike` | Name of the user. |
> | createdAt | `int64` | 1257894000 | User configuration creation time in unix.  |
> | updatedAt | `int64` | 1257894000 | User configuration update time in unix.  |
> | tags | `[]string` | ["org-tag-12345"]             | Identifiers associated with the user. |
> | keyIds | `[]string` | `["550e8400-e29b-41d4-a716-446655440000"]` | Array of key IDs.  |
> | revoked | `boolean` | `true` | Indicator for whether the user is revoked.  |
> | revokedReason | `string` | `expired` | Reason for why the user is revoked.  |
> | costLimitInUsd | `float64` | `5.5` | Total spend limit of the user.
> | costLimitInUsdOverTime | `float64` | `2` | Total spend within period of time. This field is required if costLimitInUsdUnit is specified.   |
> | costLimitInUsdUnit | `enum` | `d`                       | Time unit for costLimitInUsdOverTime. Possible values are [`m`, `h`, `d`, `mo`].      |
> | rateLimitOverTime | `int` | `2` | rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitUnit | `string` | `m`                         |  Time unit for rateLimitOverTime. Possible values are [`h`, `m`, `s`, `d`]       |
> | ttl | `string` | `24h` | time to live. Available units are [`s`, `m`, `h`] |
> | allowedPaths | `[]PathConfig` | `[{ "path": "/api/providers/openai/v1/chat/completion", "method": "POST"}]` | List of paths that can be accessed by the user. |
> | allowedModels | `[]string` | `["gpt-4"]` | List of models that can be accessed by the user.  |
> | userId | `string` | `98daa3ae-961d-4253-bf6a-322a32fdca3d` | Client defined user id. |
</details>

<details>
  <summary>Update a user via tags and user id: <code>PATCH</code> <code><b>/api/users</b></code></summary>

##### Description
This endpoint is updating a user via tags and id.

##### Query Parameters
> | name   |  type      | data type      | description                                          |
> |--------|------------|----------------|------------------------------------------------------|
> | `userId` |  optional   | `string`         | User identifier.                  |
> | `tags` |  optional  | `[]string`         | Array of identifiers associated with a user.             |

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
> | name | required | `string` | `spike` | Name of the user. |
> | keyIds | optional | `[]string` | `["98daa3ae-961d-4253-bf6a-322a32fdca3d"]` | Keys ids associated with the user. |
> | costLimitInUsd | optional | `float64` | `5.5` | Total spend limit of the user.
> | revoked | optional | `boolean` | `true` | Indicator for whether the user is revoked.  |
> | revokedReason | optional | `string` | `expired` | Reason for why the user is revoked.  |
> | costLimitInUsdOverTime | optional | `float64` | `2` | Total spend within period of time. This field is required if `costLimitInUsdUnit` is specified.   |
> | costLimitInUsdUnit | optional | `enum` | d                       | Time unit for costLimitInUsdOverTime. Possible values are [`m`, `h`, `d`, `mo`].      |
> | rateLimitOverTime | optional | `int` | 2 | Rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitUnit | optional | `enum` | m                         |  Time unit for rateLimitOverTime. Possible values are [`h`, `m`, `s`, `d`]       |
> | ttl | optional | `string` | `24h` | time to live. Available units are [`s`, `m`, `h`]. |
> | allowedPaths | optional | `[]PathConfig` | `[{ "path": "/api/providers/openai/v1/chat/completion", "method": "POST"}]` | List of paths that can be accessed by the user. |
> | allowedModels | optional | `[]string` | `["gpt-4"]` | List of models that can be accessed by the user. |

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
> | instance         | `string` | /api/users           |

##### Response
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | id | `string` | `550e8400-e29b-41d4-a716-446655440000` | Id of the user object. |
> | name | `string` | `spike` | Name of the user. |
> | createdAt | `int64` | 1257894000 | User configuration creation time in unix.  |
> | updatedAt | `int64` | 1257894000 | User configuration update time in unix.  |
> | tags | `[]string` | ["org-tag-12345"]             | Identifiers associated with the user. |
> | keyIds | `[]string` | `["550e8400-e29b-41d4-a716-446655440000"]` | Array of key IDs.  |
> | revoked | `boolean` | `true` | Indicator for whether the user is revoked.  |
> | revokedReason | `string` | `expired` | Reason for why the user is revoked.  |
> | costLimitInUsd | `float64` | `5.5` | Total spend limit of the user.
> | costLimitInUsdOverTime | `float64` | `2` | Total spend within period of time. This field is required if costLimitInUsdUnit is specified.   |
> | costLimitInUsdUnit | `enum` | `d`                       | Time unit for costLimitInUsdOverTime. Possible values are [`m`, `h`, `d`, `mo`].      |
> | rateLimitOverTime | `int` | `2` | rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitUnit | `string` | `m`                         |  Time unit for rateLimitOverTime. Possible values are [`h`, `m`, `s`, `d`]       |
> | ttl | `string` | `24h` | time to live. Available units are [`s`, `m`, `h`] |
> | allowedPaths | `[]PathConfig` | `[{ "path": "/api/providers/openai/v1/chat/completion", "method": "POST"}]` | List of paths that can be accessed by the user. |
> | allowedModels | `[]string` | `["gpt-4"]` | List of models that can be accessed by the user.  |
> | userId | `string` | `98daa3ae-961d-4253-bf6a-322a32fdca3d` | Client defined user id. |
</details>

<details>
  <summary>Get users: <code>GET</code> <code><b>/api/users</b></code></summary>

##### Description
This endpoint is retrieving users.

##### Query Parameters
> | name   |  type      | data type      | description                                          |
> |--------|------------|----------------|------------------------------------------------------|
> | `userIds` |  optional   | `[]string`         | Array of user identifiers.                  |
> | `tags` |  optional  | `[]string`         | Array of identifiers associated with a user.             |
> | `offset` |  optional  | `int`         | Pagination offset.             |
> | `limit` |  optional  | `int`         | Pagination limit.             |
> | `keyIds` |  optional  | `[]string`         | Array of key IDs associated with a user.             |

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
> | instance         | `string` | /api/users           |

##### Response
> | Response Body |
> |---------------|
> | `[]User` |

```
User
```
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | id | `string` | `550e8400-e29b-41d4-a716-446655440000` | Id of the user object. |
> | name | `string` | `spike` | Name of the user. |
> | createdAt | `int64` | 1257894000 | User configuration creation time in unix.  |
> | updatedAt | `int64` | 1257894000 | User configuration update time in unix.  |
> | tags | `[]string` | ["org-tag-12345"]             | Identifiers associated with the user. |
> | keyIds | `[]string` | `["550e8400-e29b-41d4-a716-446655440000"]` | Array of key IDs.  |
> | revoked | `boolean` | `true` | Indicator for whether the user is revoked.  |
> | revokedReason | `string` | `expired` | Reason for why the user is revoked.  |
> | costLimitInUsd | `float64` | `5.5` | Total spend limit of the user.
> | costLimitInUsdOverTime | `float64` | `2` | Total spend within period of time. This field is required if costLimitInUsdUnit is specified.   |
> | costLimitInUsdUnit | `enum` | `d`                       | Time unit for costLimitInUsdOverTime. Possible values are [`m`, `h`, `d`, `mo`].      |
> | rateLimitOverTime | `int` | `2` | rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitUnit | `string` | `m`                         |  Time unit for rateLimitOverTime. Possible values are [`h`, `m`, `s`, `d`]       |
> | ttl | `string` | `24h` | time to live. Available units are [`s`, `m`, `h`] |
> | allowedPaths | `[]PathConfig` | `[{ "path": "/api/providers/openai/v1/chat/completion", "method": "POST"}]` | List of paths that can be accessed by the user. |
> | allowedModels | `[]string` | `["gpt-4"]` | List of models that can be accessed by the user.  |
> | userId | `string` | `98daa3ae-961d-4253-bf6a-322a32fdca3d` | Client defined user id. |
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

### Images
<details>
  <summary>Generate images: <code>POST</code> <code><b>/api/providers/openai/v1/images/generations</b></code></summary>

##### Description
This endpoint is set up for generating OpenAI images. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/images/create).

</details>

<details>
  <summary>Edit images: <code>POST</code> <code><b>/api/providers/openai/v1/images/edits</b></code></summary>

##### Description
This endpoint is set up for editting OpenAI generated images. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/images/createEdit).

</details>

<details>
  <summary>Create image variations: <code>POST</code> <code><b>/api/providers/openai/v1/images/variations</b></code></summary>

##### Description
This endpoint is set up for creating OpenAI image variations. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/images/createVariation).

</details>

### Voices
<details>
  <summary>Create speech: <code>POST</code> <code><b>/api/providers/openai/v1/audio/speech</b></code></summary>

##### Description
This endpoint is set up for creating speeches. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/audio/createSpeech).

</details>

<details>
  <summary>Create transcriptions: <code>POST</code> <code><b>/api/providers/openai/v1/audio/transcriptions</b></code></summary>

##### Description
This endpoint is set up for editting generated images. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/audio/createTranscription).

</details>

<details>
  <summary>Create translations: <code>POST</code> <code><b>/api/providers/openai/v1/audios/translations</b></code></summary>

##### Description
This endpoint is set up for creating translations. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/audio/createTranslation).
</details>

### Assistants
<details>
  <summary>Create assistant: <code>POST</code> <code><b>/api/providers/openai/v1/assistants</b></code></summary>

##### Description
This endpoint is set up for creating an OpenAI assistant. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/assistants/createAssistant).

</details>

<details>
  <summary>Retrieve assistant: <code>GET</code> <code><b>/api/providers/openai/v1/assistants/:assistant_id</b></code></summary>

##### Description
This endpoint is set up for retrieving an OpenAI assistant. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/assistants/getAssistant).

</details>

<details>
  <summary>Modify assistant: <code>POST</code> <code><b>/api/providers/openai/v1/assistants/:assistant_id</b></code></summary>

##### Description
This endpoint is set up for modifying an OpenAI assistant. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/assistants/modifyAssistant).

</details>

<details>
  <summary>Delete assistant: <code>DELETE</code> <code><b>/api/providers/openai/v1/assistants/:assistant_id</b></code></summary>

##### Description
This endpoint is set up for deleting an OpenAI assistant. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/assistants/deleteAssistant).

</details>

<details>
  <summary>List assistants: <code>GET</code> <code><b>/api/providers/openai/v1/assistants</b></code></summary>

##### Description
This endpoint is set up for listing OpenAI assistants. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/assistants/listAssistants).

</details>

<details>
  <summary>Create assistant file: <code>POST</code> <code><b>/api/providers/openai/v1/assistants/:assistant_id/files</b></code></summary>

##### Description
This endpoint is set up for creating an OpenAI assistant file. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/assistants/createAssistantFile).

</details>

<details>
  <summary>Retrieve assistant file: <code>GET</code> <code><b>/api/providers/openai/v1/assistants/:assistant_id/files/:file_id</b></code></summary>

##### Description
This endpoint is set up for retrieving an OpenAI assistant file. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/assistants/getAssistantFile).

</details>

<details>
  <summary>Delete assistant file: <code>DELETE</code> <code><b>/api/providers/openai/v1/assistants/:assistant_id/files/:file_id</b></code></summary>

##### Description
This endpoint is set up for deleting an OpenAI assistant file. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/assistants/deleteAssistantFile).

</details>

<details>
  <summary>List assistant files: <code>GET</code> <code><b>/api/providers/openai/v1/assistants/:assistant_id/files</b></code></summary>

##### Description
This endpoint is set up for retrieving OpenAI assistant files. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/assistants/listAssistantFiles).

</details>

<details>
  <summary>Create thread: <code>POST</code> <code><b>/api/providers/openai/v1/threads</b></code></summary>

##### Description
This endpoint is set up for creating an OpenAI thread. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/threads/createThread).

</details>

<details>
  <summary>Retrieve thread: <code>GET</code> <code><b>/api/providers/openai/v1/threads/:thread_id</b></code></summary>

##### Description
This endpoint is set up for retrieving an OpenAI thread. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/threads/getThread).

</details>

<details>
  <summary>Modify thread: <code>POST</code> <code><b>/api/providers/openai/v1/threads/:thread_id</b></code></summary>

##### Description
This endpoint is set up for modifying an OpenAI thread. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/threads/modifyThread).

</details>

<details>
  <summary>Delete thread: <code>DELETE</code> <code><b>/api/providers/openai/v1/threads/:thread_id</b></code></summary>

##### Description
This endpoint is set up for deleting an OpenAI thread. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/threads/deleteThread).

</details>

<details>
  <summary>Create message: <code>POST</code> <code><b>/api/providers/openai/v1/threads/:thread_id/messages</b></code></summary>

##### Description
This endpoint is set up for creating an OpenAI message. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/messages/createMessage).

</details>

<details>
  <summary>Retrieve message: <code>GET</code> <code><b>/api/providers/openai/v1/threads/:thread_id/messages/:message_id</b></code></summary>

##### Description
This endpoint is set up for retrieving an OpenAI message. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/messages/getMessage).

</details>

<details>
  <summary>Modify message: <code>POST</code> <code><b>/api/providers/openai/v1/files/:file_id/content</b></code></summary>

##### Description
This endpoint is set up for modifying an OpenAI message. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/messages/modifyMessage).

</details>

<details>
  <summary>List messages: <code>GET</code> <code><b>/api/providers/openai/v1/threads/:thread_id/messages</b></code></summary>

##### Description
This endpoint is set up for listing OpenAI messages. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/messages/listMessages).

</details>

<details>
  <summary>Retrieve message file: <code>GET</code> <code><b>/api/providers/openai/v1/threads/:thread_id/messages/:message_id/files/:file_id</b></code></summary>

##### Description
This endpoint is set up for retrieving an OpenAI message file. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/messages/getMessageFile).

</details>


<details>
  <summary>List message files: <code>GET</code> <code><b>/api/providers/openai/v1/threads/:thread_id/messages/:message_id/files</b></code></summary>

##### Description
This endpoint is set up for retrieving OpenAI message files. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/messages/listMessageFiles).

</details>


<details>
  <summary>Create run: <code>POST</code> <code><b>/api/providers/openai/v1/threads/:thread_id/runs</b></code></summary>

##### Description
This endpoint is set up for creating an OpenAI run. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/runs/createRun).

</details>

<details>
  <summary>Retrieve run: <code>GET</code> <code><b>/api/providers/openai/v1/threads/:thread_id/runs/:run_id</b></code></summary>

##### Description
This endpoint is set up for retrieving an OpenAI run. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/runs/getRun).

</details>

<details>
  <summary>Modify run: <code>POST</code> <code><b>/api/providers/openai/v1/threads/:thread_id/runs/:run_id</b></code></summary>

##### Description
This endpoint is set up for modifying an OpenAI run. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/runs/modifyRun).

</details>

<details>
  <summary>List runs: <code>GET</code> <code><b>/api/providers/openai/v1/threads/runs</b></code></summary>

##### Description
This endpoint is set up for retrieving OpenAI runs. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/runs/listRuns).

</details>

<details>
  <summary>Submit tool outputs to run: <code>POST</code> <code><b>/api/providers/openai/v1/threads/runs</b></code></summary>

##### Description
This endpoint is set up for submitting tool outputs to an OpenAI run. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/runs/submitToolOutputs).

</details>

<details>
  <summary>Cancel a run: <code>POST</code> <code><b>/api/providers/openai/v1/threads/:thread_id/runs/:run_id/cancel</b></code></summary>

##### Description
This endpoint is set up for cancellling an OpenAI run. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/runs/cancelRun).

</details>

<details>
  <summary>Create thread and run: <code>POST</code> <code><b>/api/providers/openai/v1/threads/runs</b></code></summary>

##### Description
This endpoint is set up for creating an OpenAI thread and run. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/runs/createThreadAndRun).

</details>

<details>
  <summary>Retrieve run step: <code>GET</code> <code><b>/api/providers/openai/v1/threads/:thread_id/runs/:run_id/steps/:step_id</b></code></summary>

##### Description
This endpoint is set up for retrieving an OpenAI run step. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/runs/getRunStep).

</details>

<details>
  <summary>List run steps: <code>GET</code> <code><b>/api/providers/openai/v1/threads/:thread_id/runs/:run_id/steps</b></code></summary>

##### Description
This endpoint is set up for listing OpenAI run steps. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/runs/listRunSteps).

</details>

## Azure OpenAI Proxy
The custom provider proxy runs on Port `8002`.

<details>
  <summary>Create Azure OpenAI chat completion: <code>POST</code> <code><b>/api/providers/azure/openai/deployments/:deployment_id/chat/completions?api-version={API_VERSION}</b></code></summary>

##### Description
This endpoint is set up for proxying Azure OpenAI completion requests. Documentation for this endpoint can be found [here](https://learn.microsoft.com/en-us/azure/ai-services/openai/reference).

</details>

<details>
  <summary>Create Azure OpenAI embeddings: <code>POST</code> <code><b>/api/providers/azure/openai/deployments/:deployment_id/embeddings?api-version={API_VERSION}</b></code></summary>

##### Description
This endpoint is set up for proxying Azure OpenAI completion requests. Documentation for this endpoint can be found [here](https://learn.microsoft.com/en-us/azure/ai-services/openai/reference).

</details>


## Anthropic Proxy
The custom provider proxy runs on Port `8002`.

<details>
  <summary>Create Anthropic completion: <code>POST</code> <code><b>/api/providers/anthropic/v1/complete</b></code></summary>

##### Description
This endpoint is set up for proxying Anthropic completion requests. Documentation for this endpoint can be found [here](https://docs.anthropic.com/claude/reference/complete_post).

</details>

<details>
  <summary>Create Anthropic messages: <code>POST</code> <code><b>/api/providers/anthropic/v1/messages</b></code></summary>

##### Description
This endpoint is set up for proxying Anthropic messages requests. Documentation for this endpoint can be found [here](https://docs.anthropic.com/claude/reference/messages_post).

</details>

## vllm Provider Proxy
The vllm provider proxy runs on Port `8002`.

<details>
  <summary>Create chat completions: <code>POST</code> <code><b>/api/providers/vllm/v1/chat/completions</b></code></summary>

##### Description
This endpoint is set up for proxying vllm chat completions requests. Documentation for this endpoint can be found [here](https://docs.vllm.ai/en/latest/serving/openai_compatible_server.html).

</details>

<details>
  <summary>Create completions: <code>POST</code> <code><b>/api/providers/vllm/v1/completions</b></code></summary>

##### Description
This endpoint is set up for proxying vllm completions requests. Documentation for this endpoint can be found [here](https://docs.vllm.ai/en/latest/serving/openai_compatible_server.html).

</details>

## Deepinfra Provider Proxy
The deepinfra provider proxy runs on Port `8002`.

<details>
  <summary>Create chat completions: <code>POST</code> <code><b>/api/providers/deepinfra/v1/chat/completions</b></code></summary>

##### Description
This endpoint is set up for proxying deepinfra chat completions requests. Documentation for this endpoint can be found [here](https://deepinfra.com/docs/advanced/openai_api).

</details>

<details>
  <summary>Create completions: <code>POST</code> <code><b>/api/providers/deepinfra/v1/completions</b></code></summary>

##### Description
This endpoint is set up for proxying deepinfra completions requests. Documentation for this endpoint can be found [here](https://deepinfra.com/docs/advanced/openai_api).

</details>

<details>
  <summary>Create embeddings: <code>POST</code> <code><b>/api/providers/deepinfra/v1/embeddings</b></code></summary>

##### Description
This endpoint is set up for proxying deepinfra embeddings requests. Documentation for this endpoint can be found [here](https://deepinfra.com/docs/advanced/openai_api).

</details>

## Custom Provider Proxy
The custom provider proxy runs on Port `8002`.

<details>
  <summary>Call custom providers: <code>POST</code> <code><b>/api/custom/providers/:provider/*</b></code></summary>

##### Description
First you need to use create custom providers endpoint to create custom providers. Then create corresponding provider setting for the newly created custom provider. Afterward, you can start creating keys associated with the custom provider, and use the keys to access this endpoint by placing the created key in ```Authorization: Bearer YOUR_BRICKSLLM_KEY``` as part of your HTTP request headers.

</details>

## Route Proxy
The custom provider proxy runs on Port `8002`.

<details>
  <summary>Call a route: <code>POST</code> <code><b>/api/route/*</b></code></summary>

##### Description
Route helps you interpolate different models (embeddings or chat completion models) and providers (OpenAI or Azure OpenAI) to gurantee API responses.

First you need to use create route endpoint to create routes. If the route uses both Azure and OpenAI, you need to create API keys with corresponding provider settings as well. If the route is for chat completion, just call the route using the [OpenAI chat completion format](https://platform.openai.com/docs/api-reference/chat). On the other hand, if the route is for embeddings, just call the route using the [embeddings format](https://platform.openai.com/docs/api-reference/embeddings).
 
</details>

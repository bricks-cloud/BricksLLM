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

**BricksLLM** is a cloud native AI gateway written in Go. Currently it is a proxy for OpenAI. The main feature of the proxy is letting you create API keys that has a rate limit, cost limit and ttl that you could use in both development and production use cases to achieve fine-grained access control that is not provided by OpenAI at the moment. The proxy is compatible with OpenAI API and its SDKs. 

The vision of BricksLLM is to support many more large language models such as LLama2, Claude, PaLM2 etc, and streamline LLM operations.

## Roadmap
* [x] Access control via API key with rate limit, cost limit and ttl
* Statsd integration
* Logging integration
* Routes configuration
* PII detection and masking


# Installation
## Prerequisites
* [go 1.19+](https://go.dev/dl/)
* [Docker](https://www.docker.com/get-started/)

## Getting Started
BricksLLM API gateway uses postgresql to store configurations, and redis for caching. Therefore, they are required for running BricksLLM.

Setting up postgresql and redis via docker-compose.

```bash
docker-compose up -d
```

Setting up your OpenAI API credential
```bash
export OPENAI_API_KEY = "YOUR_OPENAI_API_CREDENTIAL"
export POSTGRESQL_USERNAME = "YOUR_POSTGRESQL_USERNAME"
export POSTGRESQL_PASSWORD = "YOUR_POSTGRESQL_PASSWORD"
export REDIS_PASSWORD = "YOUR_REDIS_PASSWORD"
```

Spinning up the proxy server by runnning
```bash
go run ./cmd/tool/main.go
```

# Documentation
## Environment variables
> | Name | type | description | default
> |---------------|-----------------------------------|----------|
> | `OPENAI_API_KEY`         | required | OpenAI API Key|
> | `POSTGRESQL_HOSTS`       | optional | Hosts for Postgresql DB. Seperated by , | localhost
> | `POSTGRESQL_USERNAME`         | required | Postgresql DB username|
> | `POSTGRESQL_PASSWORD`         | required | Postgresql DB password|
> | `POSTGRESQL_SSL_ENABLED`         | optional | Postgresql SSL enabled| ```false```
> | `POSTGRESQL_PORT`         | optional | The port that Postgresql DB runs on| ```5432```
> | `POSTGRESQL_READ_TIME_OUT`         | optional | Timeout for Postgresql read operations | ```2s```
> | `POSTGRESQL_WRITE_TIME_OUT`         | optional | Timeout for Postgresql write operations | ```1s```
> | `REDIS_PASSWORD`         | required | Password for |
> | `REDIS_PORT`         | optional | Timeout for Redis read operations | 
> | `REDIS_READ_TIME_OUT`         | optional | Timeout for Redis read operations | ```1s```
> | `REDIS_WRITE_TIME_OUT`         | optional | Timeout for Redis write operations | ```500ms```
> | `IN_MEMORY_DB_UPDATE_INTERVAL`         | optional | The interval BricksLLM API gateway polls Postgresql DB for latest key configurations | ```10s```

## Configuration Endpoints
The configuration server runs on Port ```8001```.
<details>
  <summary><code>GET</code> <code><b>/api/key-management/keys?tag={tag}</b></code></summary>

##### Description
This endpoint is set up for retrieving key configurations using a query param called tag.

##### Parameters

> | name   |  type      | data type      | description                                          |
> |--------|------------|----------------|------------------------------------------------------|
> | `tag` |  required  | string         | Identifier attached to a key configuration                  |

##### Error Response

> | http code     | content-type                      |
> |---------------|-----------------------------------|
> | `400`, `500`         | `application/json`                |

> | Field     | type | example                      |
> |---------------|-----------------------------------|-|
> | status         | ```number``` | 400            |
> | title         | ```string``` | request body reader error             |
> | type         | ```string``` | /errors/request-body-read             |
> | detail         | ```string``` | something is wrong            |
> | instance         | ```string``` | /api/key-management/keys            |

##### Response

> | Response Body |
> |---------------|
> | ```[]KeyConfiguration``` |

Fields of KeyConfiguration
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | name | ```string``` | spike's developer key | Name of the API key. |
> | createdAt | ```number``` | 1257894000 | Key configuration creation time in unix.  |
> | updatedAt | ```number``` | 1257894000 | Key configuration update time in unix.  |
> | revoked | ```boolean``` | true | Indicator for whether the key is revoked.  |
> | revokedReason | ```string``` | The key has expired | Reason for why the key is revoked.  |
> | tags | ```[]string``` | ["org-tag-12345"]             | Identifiers associated with the key. |
> | keyId | ```string``` | 550e8400-e29b-41d4-a716-446655440000 | Unique identifier for the key.  |
> | costLimitInUsd | ```number``` | 5.5 | Total spend limit of the API key.
> | costLimitInUsdOverTime | ```string``` | 2 | Total spend within period of time. This field is required if costLimitInUsdUnit is specified.   |
> | costLimitInUsdUnit | ```enum``` | d                       | Time unit for costLimitInUsdOverTime. Possible values are [`h`, `m`, `s`, `d`].      |
> | rateLimitOverTime | ```string``` | 2 | rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitOverTime | ```string``` | 2 | rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitUnit | ```string``` | m                         |  Time unit for rateLimitOverTime. Possible values are [`h`, `m`, `s`, `d`]       |
> | ttl | ```string``` | 2d | time to live. |

</details>


<details>
  <summary><code>PUT</code> <code><b>/api/key-management/keys</b></code></summary>

##### Description
This endpoint is set up for retrieving key configurations using a query param called tag.

##### Request
> | Field | type | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | name | required | ```string``` | spike's developer key | Name of the API key. |
> | tags | optional | ```[]string``` | ["org-tag-12345"]             | Identifiers associated with the key. |
> | key | required | ```string``` | abcdef12345 | API key |
> | costLimitInUsd | optional | ```number``` | 5.5 | Total spend limit of the API key.
> | costLimitInUsdOverTime | optional | ```string``` | 2 | Total spend within period of time. This field is required if costLimitInUsdUnit is specified.   |
> | costLimitInUsdUnit | optional | ```enum``` | d                       | Time unit for costLimitInUsdOverTime. Possible values are [`h`, `m`, `s`, `d`].      |
> | rateLimitOverTime | optional | ```string``` | 2 | rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitUnit | optional | ```enum``` | m                         |  Time unit for rateLimitOverTime. Possible values are [`h`, `m`, `s`, `d`]       |
> | ttl | optional | ```string``` | 2d | time to live. |

##### Error Response

> | http code     | content-type                      |
> |---------------|-----------------------------------|
> | `400`, `500`         | `application/json`                |

> | Field     | type | example                      |
> |---------------|-----------------------------------|-|
> | status         | ```number``` | 400            |
> | title         | ```string``` | request body reader error             |
> | type         | ```string``` | /errors/request-body-read             |
> | detail         | ```string``` | something is wrong            |
> | instance         | ```string``` | /api/key-management/keys            |

##### Responses
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | name | ```string``` | spike's developer key | Name of the API key. |
> | createdAt | ```number``` | 1257894000 | Key configuration creation time in unix.  |
> | updatedAt | ```number``` | 1257894000 | Key configuration update time in unix.  |
> | revoked | ```boolean``` | true | Indicator for whether the key is revoked.  |
> | revokedReason | ```string``` | The key has expired | Reason for why the key is revoked.  |
> | tags | ```[]string``` | ["org-tag-12345"]             | Identifiers associated with the key. |
> | keyId | ```string``` | 550e8400-e29b-41d4-a716-446655440000 | Unique identifier for the key.  |
> | costLimitInUsd | ```number``` | 5.5 | Total spend limit of the API key.
> | costLimitInUsdOverTime | ```string``` | 2 | Total spend within period of time. This field is required if costLimitInUsdUnit is specified.   |
> | costLimitInUsdUnit | ```enum``` | d                       | Time unit for costLimitInUsdOverTime. Possible values are [`h`, `m`, `s`, `d`].      |
> | rateLimitOverTime | ```string``` | 2 | rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitOverTime | ```string``` | 2 | rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitUnit | ```string``` | m                         |  Time unit for rateLimitOverTime. Possible values are [`h`, `m`, `s`, `d`]       |
> | ttl | ```string``` | 2d | time to live. |

</details>

<details>
  <summary><code>PATCH</code> <code><b>/api/key-management/keys/{keyId}</b></code></summary>

##### Description
This endpoint is set up for updating key configurations using key id.

##### Parameters
> | name   |  type      | data type      | description                                          |
> |--------|------------|----------------|------------------------------------------------------|
> | `keyId` |  required  | string         | Unique key configuration identifier.                  |

##### Request
> | Field | type | type | example                      | description |
> |---------------|-----------------------------------|-|-|-|
> | name | optional | ```string``` | spike's developer key | Name of the API key. |
> | tags | optional | ```[]string``` | ["org-tag-12345"]             | Identifiers associated with the key. |
> | revoked | optional |  ```boolean``` | true | Indicator for whether the key is revoked.  |
> | revokedReason| optional | ```string``` | The key has expired | Reason for why the key is revoked.  |
> | costLimitInUsdOverTime | optional | ```string``` | 2 | Total spend within period of time. This field is required if costLimitInUsdUnit is specified.   |
> | costLimitInUsdUnit | optional | ```enum``` | d                       | Time unit for costLimitInUsdOverTime. Possible values are [`h`, `m`, `s`, `d`].      |
> | rateLimitOverTime | optional | ```string``` | 2 | rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitUnit | optional | ```enum``` | m                         |  Time unit for rateLimitOverTime. Possible values are [`h`, `m`, `s`, `d`]       |

##### Error Response

> | http code     | content-type                      |
> |---------------|-----------------------------------|
> | `400`, `500`         | `application/json`                |

> | Field     | type | example                      |
> |---------------|-----------------------------------|-|
> | status         | ```number``` | 400            |
> | title         | ```string``` | request body reader error             |
> | type         | ```string``` | /errors/request-body-read             |
> | detail         | ```string``` | something is wrong            |
> | instance         | ```string``` | /api/key-management/keys            |

##### Response
> | Field | type | example                      | description |
> |---------------|-----------------------------------|-|-|
> | name | ```string``` | spike's developer key | Name of the API key. |
> | createdAt | ```number``` | 1257894000 | Key configuration creation time in unix.  |
> | updatedAt | ```number``` | 1257894000 | Key configuration update time in unix.  |
> | revoked | ```boolean``` | true | Indicator for whether the key is revoked.  |
> | revokedReason | ```string``` | The key has expired | Reason for why the key is revoked.  |
> | tags | ```[]string``` | ["org-tag-12345"]             | Identifiers associated with the key. |
> | keyId | ```string``` | 550e8400-e29b-41d4-a716-446655440000 | Unique identifier for the key.  |
> | costLimitInUsd | ```number``` | 5.5 | Total spend limit of the API key.
> | costLimitInUsdOverTime | ```string``` | 2 | Total spend within period of time. This field is required if costLimitInUsdUnit is specified.   |
> | costLimitInUsdUnit | ```enum``` | d                       | Time unit for costLimitInUsdOverTime. Possible values are [`h`, `m`, `s`, `d`].      |
> | rateLimitOverTime | ```string``` | 2 | rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitOverTime | ```string``` | 2 | rate limit over period of time. This field is required if rateLimitUnit is specified.    |
> | rateLimitUnit | ```string``` | m                         |  Time unit for rateLimitOverTime. Possible values are [`h`, `m`, `s`, `d`]       |
> | ttl | ```string``` | 2d | time to live. |

</details>

## OpenAI Proxy
The OpenAI proxy runs on Port ```8002```.

<details>
  <summary><code>POST</code> <code><b>/api/providers/openai</b></code></summary>

##### Description
This endpoint is set up for proxying OpenAI API requests. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/chat).

</details>
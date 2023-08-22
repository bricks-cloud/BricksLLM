<p align="center">
<img src="./assets/bricks-logo.png" width="150" />
</p>

# **BricksLLM: API gateway For running LLM applications in Production**

<p align="center">
   <a href='https://www.ycombinator.com/'><img alt='YCombinator S22' src='https://img.shields.io/badge/Y%20Combinator-2022-orange'/></a>
   <a href='http://makeapullrequest.com'><img alt='PRs Welcome' src='https://img.shields.io/badge/PRs-welcome-43AF11.svg?style=shields'/></a>
   <a href="https://discord.gg/dFvdt4wqWh"><img src="https://img.shields.io/badge/discord-BricksLLM-blue?logo=discord&labelColor=2EB67D" alt="Join BricksLLM on Discord"></a>
   <a href="https://github.com/bricks-cloud/bricks/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-red" alt="License"></a>
</p>

**BricksLLM** is a cloud native API gateway written in Go. It provides the following features:

* API key authentication
* Rate limit
* LLM spend limit
* LLM input and output logging with privacy control(:construction:)
* Statsd integration (:construction:)
* LLM prompt AB testing (:construction:)
* PII detection and masking (:construction:)
* Support for more LLM models (:construction:)

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
```

Spinning up the proxy server by runnning
```bash
go run ./cmd/tool/main.go
```

# Documentation
## Environment variables
> | Name | description | example
> |---------------|-----------------------------------|----------|
> | `OPENAI_API_KEY`         | OpenAI API Key| YOUR_API_KEY
> | `POSTGRESQL_HOSTS`       | Hosts for Postgresql DB. Seperated by , | localhost
> | `POSTGRESQL_USERNAME`         | Postgresql DB username| username
> | `POSTGRESQL_PASSWORD`         | Postgresql DB password| password
> | `POSTGRESQL_SSL_ENABLED`         | Postgresql SSL enabled| ```true```
> | `POSTGRESQL_PORT`         | The port that Postgresql DB runs on| ```5432```
> | `POSTGRESQL_READ_TIME_OUT`         | Timeout for Postgresql read operations | ```2s```
> | `POSTGRESQL_WRITE_TIME_OUT`         | Timeout for Postgresql write operations | ```1s```
> | `REDIS_READ_TIME_OUT`         | Timeout for Redis read operations | ```1s```
> | `REDIS_READ_WRITE_OUT`         | Timeout for Redis write operations | ```500ms```
> | `IN_MEMORY_DB_UPDATE_INTERVAL`         | The interval BricksLLM API gateway polls Postgresql DB for latest key configurations | ```10s```

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

```[]KeyConfiguration```

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
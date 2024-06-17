<p align="center">
<img src="./assets/bricks-logo.png" width="150" />
</p>

# **BricksLLM: AI Gateway For Putting LLMs In Production**

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
> | `POSTGRESQL_SSL_MODE`         | optional | Postgresql SSL mode| `disable` |
> | `POSTGRESQL_PORT`         | optional | The port that Postgresql DB runs on| `5432` |
> | `POSTGRESQL_READ_TIME_OUT`         | optional | Timeout for Postgresql read operations | `2m` |
> | `POSTGRESQL_WRITE_TIME_OUT`         | optional | Timeout for Postgresql write operations | `5s` |
> | `REDIS_HOSTS`         | required | Host for Redis. Separated by , | `localhost` |
> | `REDIS_PASSWORD`         | optional | Redis Password |
> | `REDIS_PORT`         | optional | The port that Redis DB runs on | `6379` |
> | `REDIS_READ_TIME_OUT`         | optional | Timeout for Redis read operations | `1s` |
> | `REDIS_WRITE_TIME_OUT`         | optional | Timeout for Redis write operations | `500ms` |
> | `IN_MEMORY_DB_UPDATE_INTERVAL`         | optional | The interval BricksLLM API gateway polls Postgresql DB for latest key configurations | `1s` |
> | `STATS_PROVIDER`         | optional | This value can only be datadog. Required for integration with Datadog.  |
> | `PROXY_TIMEOUT`         | optional | Timeout for proxy HTTP requests. | `600s` |
> | `NUMBER_OF_EVENT_MESSAGE_CONSUMERS`         | optional | Number of event message consumers that help handle counting tokens and inserting event into db.  | `3` |
> | `AWS_SECRET_ACCESS_KEY`         | optional | It is for PII detection feature.  | `5s` |
> | `AWS_ACCESS_KEY_ID`         | optional | It is for using PII detection feature.  | `5s` |
> | `AMAZON_REGION`         | optional | Region for AWS.  | `us-west-2` |
> | `AMAZON_REQUEST_TIMEOUT`         | optional | Timeout for amazon requests.  | `5s` |
> | `AMAZON_CONNECTION_TIMEOUT`         | optional | Timeout for amazon connection.  | `10s` |
> | `ADMIN_PASS`         | optional | Simple password for the admin server. |

## Admin Server
[Swagger Doc](https://bricks-cloud.github.io/BricksLLM/admin)

## Proxy Server
[Swagger Doc](https://bricks-cloud.github.io/BricksLLM/proxy)


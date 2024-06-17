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

## Configuration Endpoints
The configuration server runs on Port `8001`.

##### Headers
> | name   |  type      | data type      | description                                          |
> |--------|------------|----------------|------------------------------------------------------|
> | `X-API-KEY` |  optional  | `string`         | Key authentication header.

##### Documentation
[Swagger Doc](https://bricks-cloud.github.io/BricksLLM/#/)

# Proxy Health Check
<details>
  <summary>Health Check: <code>GET</code> <code><b>/api/health</b></code></summary>

  ##### Response
> | http code     |
> |---------------|
> | `200`         |
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

### Batches
<details>
  <summary>Create a batch: <code>POST</code> <code><b>/api/providers/openai/v1/batches</b></code></summary>

##### Description
This endpoint is set up for creating a batch. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/batch/create).

</details>

<details>
  <summary>Retrieve a batch: <code>GET</code> <code><b>/api/providers/openai/v1/batches/:batch_id</b></code></summary>

##### Description
This endpoint is set up for retrieving a batch. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/batch/retrieve).

</details>

<details>
  <summary>Cancel a batch: <code>POST</code> <code><b>/api/providers/openai/v1/batches/:batch_id/cancel</b></code></summary>

##### Description
This endpoint is set up for canceling a batch. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/batch/cancel).

</details>

<details>
  <summary>List batches: <code>GET</code> <code><b>/api/providers/openai/v1/batches</b></code></summary>

##### Description
This endpoint is set up for listing batches. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/batch/list).

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
This endpoint is set up for editing OpenAI generated images. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/images/createEdit).

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
This endpoint is set up for editing generated images. Documentation for this endpoint can be found [here](https://platform.openai.com/docs/api-reference/audio/createTranscription).

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

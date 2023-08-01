<p align="center">
<img src="./assets/bricks-logo.png" width="150" />
</p>

# **BricksLLM: A Declarative Approach To Building LLM Applications**

<p align="center">
   <a href='http://makeapullrequest.com'><img alt='PRs Welcome' src='https://img.shields.io/badge/PRs-welcome-43AF11.svg?style=shields'/></a>
   <a href="https://discord.gg/dFvdt4wqWh"><img src="https://img.shields.io/badge/discord-BricksLLM-blue?logo=discord&labelColor=2EB67D" alt="Join BricksLLM on Discord"></a>
   <a href="https://github.com/bricks-cloud/bricks/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-red" alt="License"></a>
</p>

**BricksLLM** is a declarative Go framework that gives you building blocks to create reliable LLM workflows. Productionizing LLM applications is difficult due to the technology's probalistic nature. **BricksLLM** solves this issue by accelerating the developer development cycle and allowing developers to relentlessly test and improve their LLM applications continuously.

* **Build**: Use BricksLLM building blocks to quickly build out your LLM backend.
* **Test**: Unit test prompts and BricksLLM APIs in CI/CD.
* **Version Control**: Version control your prompt strategies through declaractive configuration.
* **Deploy**: Containerize and deploy BricksLLM anywhere.
* **Monitor**: Out of the box detailed logging and monitoring.
* **A/B Testing**: Fine tune your prompt strategies through A/B testing.

## Overview
Here is an example of BricksLLM config yaml.

```yaml
openai:
  api_credential: ${OPENAI_KEY}

routes:
  - path: /travel
    provider: openai
    key_auth:
      key: ${API_KEY}
    cors:
      allowed_origins: ["*"]
      allowed_credentials: true
    input:
      plan:
        type: object
        properties:
          place:
            type: string
    openai_config:
      model: gpt-3.5-turbo
      prompts:
        - role: assistant
          content: say hi to {{ plan.place }}

  - path: /test
    provider: openai
    input:
      name:
        type: string
    openai_config:
      model: gpt-3.5-turbo
      prompts:
        - role: assistant
          content: say hi to {{ name }}

```
### Core Components
Each BricksLLM application has to have at least one route configuration

```yaml
routes:
  - path: /travel
    provider: openai
    key_auth:
      key: ${API_KEY}
    cors:
      allowed_origins: ["*"]
      allowed_credentials: true
    input:
      plan:
        type: object
        properties:
          place:
            type: string
    openai_config:
      model: gpt-3.5-turbo
      prompts:
        - role: assistant
          content: say hi to {{ plan.place }}
```

### Observability Components
There are also observability components such as `logger` that let you control how BricksLLM exposes log data.

```yaml
logger:
  api:
    hide_ip: true
    hide_headers: true
  llm:
    hide_headers: true
    hide_response_content: true
    hide_prompt_content: true
```

### Resource Components
Resource components let you specify available resources that could be used in the core components

```yaml
openai:
  api_credential: ${OPENAI_KEY}
```

# Documentation
## routes
### Example
```yaml
routes:
  - path: /travel
    provider: openai
    key_auth:
      key: ${API_KEY}
    cors:
      allowed_origins: ["*"]
      allowed_credentials: true
    input:
      plan:
        type: object
        properties:
          place:
            type: string
    openai_config:
      model: gpt-3.5-turbo
      prompts:
        - role: assistant
          content: say hi to {{ plan.place }}
```
### Fields
#### `routes` 
##### Required: ```true```
##### Type: ```array```
A list of route configurations connected with LLM APIs.

#### `routes[].path`
##### Required: ```true```
##### Type: ```string```
A path specifies the resource that gets exposed to the client.

```yaml
routes:
  - path: /weathers
```

#### `routes[].provider`
##### Required: ```true```
##### Type: ```enum```
##### Options: [`openai`] 
A provider is the name of the service that provides the LLM API. Right now, Bricks only supports OpenAI.

#### `routes[].key_auth`
##### Required: ```false```
##### Type: ```object```
Contains configurations for using API key authentication. If set up, BricksLLM would start checking API header `X-Api-Key` of incoming request for authentication and return ```401``` if the API call is unauthorized.

#### `routes[].key_auth.key`
##### Required: `true`
##### Type: `string`
A unique key to authenticate the API.

#### `routes[].cors`
##### Required: `false`
##### Type: `object`
The `cors` field is used to specify Cross-Origin Resource Sharing settings. It includes subfields for setting up CORS policies.

#### `routes[].cors.allowed_origins`
##### Required: `true`
##### Type: `array`
An array of strings specifying allowed origins for CORS.

#### `routes[].cors.allowed_credentials`
##### Required: `true`
##### Type: `boolean`
A boolean value specifying if CORS response can include credentials. It sets `Access-Control-Allow-Credentials` to `true` in server responses. You can read more about it [here](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Credentials).

#### `routes[].input`
##### Required: `false`
##### Type: `object`
The `input` field is used to specify the input JSON schema of the route.

```yaml
    input:
      plan:
        type: object
        properties:
          place:
            type: string

```
This translates to the following JSON schema.

```json
{
    "plan": {
        "type": "",
    }
}
```
BricksLLM would use this schema to validate incoming HTTP requests. It would return `400` if expected fields in the request body are empty or the data type does not match the schema.

#### `routes[].input[field_name].type`
##### Required: `false`
##### Type: `enum`
##### Options: [`string`, `boolean`, `object`, `number`]
The `type` field within input specifies the data type of the expected

#### `routes[].input[field_name].properties`
##### Required: `true` (Only if the field data type is `object`)
##### Type: `object`
The `properties` field specifies the schema of the `object` field.

#### `routes[].openai_config`
##### Required: `false`
##### Type: `object`
The `openai_config` field is used to specify the configuration details for OpenAI.

#### `routes[].openai_config.api_credential`
##### Required: `true` (Only required `openai.api_credential` is not specified)
##### Type: `string`
OpenAI API credential. If `openai.api_credential` is specified, the credential here will overwrite it in the API call to OpenAI.

```yaml
    openai_config:
      model: gpt-3.5-turbo
      api_credential: ${OPENAI_KEY}
```

:warning: **Store OpenAI key securely, and only use it as an environment variable.**

#### `routes[].openai_config.model`
##### Required: `true`
##### Type: `enum`
##### Options: [`gpt-3.5-turbo`, `gpt-3.5-turbo-16k`, `gpt-3.5-turbo-0613`, `gpt-3.5-turbo-16k-0613`, `gpt-4`, `gpt-4-0613`, `gpt-4-32k`, `gpt-4-32k-0613`]

The `model` field specifies the version of the model to use. Right now, BricksLLM supports all `gpt-3.5` and `gpt-4` OpenAI models.

#### `routes[].openai_config.prompts`
##### Required: `true`
##### Type: `array`
An array of objects that define how the model should respond to input.

#### `routes[].openai_config.prompts[].role`
##### Required: `true`
##### Type: `enum`
##### Options: [`assisstant, system, user`] 
The `role` field specifies the role the model should take when responding to input.

#### `routes[].openai_config.prompts[].content`
##### Required: `true`
##### Type: `string`
The `content` field specifies content of the prompt.

## openai
### Example
```yaml
openai:
  api_credential: ${OPENAI_KEY}
```

### Fields
#### `openai`
##### Required: `true`
##### Type: `string`
It contains the configuration of OpenAI API.

#### `openai.api_credential`
##### Required: `true`
##### Type: `string`
OpenAI API credential. If `routes[].openai_config.api_credential` is specified, it will overwrite this credential in the OpenAI API call.

:warning: **Store OpenAI key securely, and only use it as an environment variable.**

## logger
### Example
```yaml
logger:
  api:
    hide_ip: true
    hide_headers: true
  llm:
    hide_headers: true
    hide_response_content: true
    hide_prompt_content: true
```

### Fields
#### `logger`
##### Required: `false`
##### Type: `object`
It contains the configuration of the logger.

#### `logger.api`
##### Required: `false`
##### Type: `object`
It contains the configuration for API log. Here is an example of the API log:
```json
{
    "clientIp": "",
    "instanceId": "fcfc0aa8-0f57-4d24-82f9-247b68e4dcaf",
    "latency": {
        "proxy": 886,
        "bricksllm": 3,
        "total": 889
    },
    "created_at": 1690773969,
    "route": {
        "path": "/travel",
        "protocol": "http"
    },
    "response": {
        "headers": {
            "Access-Control-Allow-Credentials": [
                "true"
            ],
            "Access-Control-Allow-Origin": [
                "https://figma.com"
            ]
        },
        "createdAt": 1690773970,
        "status": 200,
        "size": 295
    },
    "request": {
        "headers": {
            "Accept": [
                "*/*"
            ],
            "Accept-Encoding": [
                "gzip, deflate, br"
            ],
            "Connection": [
                "keep-alive"
            ],
            "Content-Length": [
                "50"
            ],
            "Content-Type": [
                "application/json"
            ],
            "Origin": [
                "https://figma.com"
            ],
            "Postman-Token": [
                "d05b5651-542e-4a4d-9fa1-f00d005488ad"
            ],
            "User-Agent": [
                "PostmanRuntime/7.32.2"
            ]
        },
        "size": 50
    },
    "type": "api"
}
```


#### `logger.llm`
##### Required: `false`
##### Type: `object`
It contains the configuration for LLM API log. Here is an example of the LLM API log. 
```json
{
    "instanceId": "fcfc0aa8-0f57-4d24-82f9-247b68e4dcaf",
    "type": "llm",
    "token": {
        "prompt_tokens": 12,
        "completion_tokens": 10,
        "total": 22
    },
    "response": {
        "id": "chatcmpl-7iDpqAjqdAp1AeBT6ZVRpYrgrMQ3z",
        "headers": {
            "Access-Control-Allow-Origin": [
                "*"
            ],
            "Alt-Svc": [
                "h3=\":443\"; ma=86400"
            ],
            "Cache-Control": [
                "no-cache, must-revalidate"
            ],
            "Cf-Cache-Status": [
                "DYNAMIC"
            ],
            "Cf-Ray": [
                "7ef2bcffd9dd173a-SJC"
            ],
            "Content-Type": [
                "application/json"
            ],
            "Date": [
                "Mon, 31 Jul 2023 03:26:10 GMT"
            ],
            "Openai-Model": [
                "gpt-3.5-turbo-0613"
            ],
            "Openai-Organization": [
                "acme"
            ],
            "Openai-Processing-Ms": [
                "560"
            ],
            "Openai-Version": [
                "2020-10-01"
            ],
            "Server": [
                "cloudflare"
            ],
            "Strict-Transport-Security": [
                "max-age=15724800; includeSubDomains"
            ],
            "X-Ratelimit-Limit-Requests": [
                "3500"
            ],
            "X-Ratelimit-Limit-Tokens": [
                "90000"
            ],
            "X-Ratelimit-Remaining-Requests": [
                "3499"
            ],
            "X-Ratelimit-Remaining-Tokens": [
                "89977"
            ],
            "X-Ratelimit-Reset-Requests": [
                "17ms"
            ],
            "X-Ratelimit-Reset-Tokens": [
                "14ms"
            ],
            "X-Request-Id": [
                "200f64b5d1ad324c5293dba96132b31c"
            ]
        },
        "created_at": 1690773970,
        "size": 435,
        "status": 200,
        "choices": [
            {
                "role": "assistant",
                "content": "Hi Beijing! How can I assist you today?",
                "finish_reason": "stop"
            }
        ]
    },
    "request": {
        "headers": {
            "Content-Type": [
                "application/json"
            ]
        },
        "model": "gpt-3.5-turbo",
        "messages": [
            {
                "role": "assistant",
                "content": "say hi to beijing"
            }
        ],
        "size": 89,
        "created_at": 1690773969
    },
    "provider": "openai",
    "estimated_cost": 0.038000000000000006,
    "created_at": 1690773969,
    "latency": 886
}
```

#### `logger.api.hide_ip`
##### Required: `false`
##### Default: `false`
##### Type: `boolean`
This field prevents logger from logging the http request ip.

#### `logger.api.hide_headers`
##### Required: `false`
##### Default: `false`
##### Type: `boolean`
This field prevents logger from logging the http request and response headers.


#### `logger.llm.hide_headers`
##### Required: `false`
##### Default: `false`
##### Type: `boolean`
This field prevents logger from logging the upstream llm http request and llm response headers.

#### `logger.llm.hide_response_content`
##### Required: `false`
##### Default: `false`
##### Type: `boolean`
This field prevents logger from logging the upstream llm http response content.

#### `logger.llm.hide_prompt_content`
##### Required: `false`
##### Default: `false`
##### Type: `boolean`
This field prevents logger from logging the upstream llm http request prompt content.
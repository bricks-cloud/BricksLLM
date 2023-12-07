# Granular Access Control Guide

## Getting Started
This guide shows you how to create API keys with more granular access control. Let's say you want to create API keys that limits the user to only OpenAI chat completion endpoint with the model ```gpt-3.5-turbo```.

### Step 1 - Create a provider
```bash
curl -X PUT http://localhost:8001/api/provider-settings \
   -H "Content-Type: application/json" \
   -d '{
          "provider":"openai",
          "setting": {
             "apikey": "YOUR_OPENAI_KEY"
          },
          "allowedModels": ["gpt-3.5-turbo"]
      }'   
```
Copy the `id` from the response.

### Step 2 - Create a Bricks API key
Use `id` from the previous step as `settingId` to create a key with a rate limit of 2 req/min and a spend limit of 25 cents that is only allowed to use the model ```gpt-3.5-turbo``` for chat completion endpoint.
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
        "costLimitInUsd": 0.25,
        "allowedPaths": [{
            "path": "/api/providers/openai/v1/chat/completions",
            "method": "POST"
        }]
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

Trying to access a different path will result in ```403```.

```bash
curl -X POST http://localhost:8002/api/providers/openai/v1/embeddings \
   -H "Authorization: Bearer my-secret-key" \
   -H "Content-Type: application/json" \
   -d '{
        "input": "The food was delicious and the waiter...",
        "model": "text-embedding-ada-002",
        "encoding_format": "float"
    }'
```

```json
{
    "error": {
        "code": 403,
        "message": "[BricksLLM] path is not allowed",
        "type": ""
    }
}
```

Trying to access a different model will also result in ```403```.

```bash
curl -X POST http://localhost:8002/api/providers/openai/v1/chat/completions \
   -H "Authorization: Bearer my-secret-key" \
   -H "Content-Type: application/json" \
   -d '{
          "model": "gpt-4",
          "messages": [
              {
                  "role": "system",
                  "content": "hi"
              }
          ]
      }'
```

```json
{
    "error": {
        "code": 403,
        "message": "[BricksLLM] model is not allowed",
        "type": ""
    }
}
```
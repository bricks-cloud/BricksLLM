# vLLM integration Guide

## Getting Started
This guide shows you how to create API keys for vLLM integration. Let's say you want to create API for accessing the model ```facebook/opt-125m```.


### Step 1 - Create a provider
```bash
curl -X PUT http://localhost:8001/api/provider-settings \
   -H "Content-Type: application/json" \
   -d '{
          "provider":"vllm",
          "setting": {
             "url": "YOUR_VLLM_DEPLOYMENT_URL",
             "apikey": "YOUR_VLLM_API_KEY"
          }
      }'   
```
Copy the `id` from the response. `YOUR_VLLM_DEPLOYMENT_URL` should not have `/` at the end.

### Step 2 - Create a Bricks API key
Use `id` from the previous step as `settingId` to create a key with a rate limit of 2 req/min and a spend limit of 25 cents.
```bash
curl -X PUT http://localhost:8001/api/key-management/keys \
   -H "Content-Type: application/json" \
   -d '{
	      "name": "My vLLM Key",
	      "key": "my-vllm-key",
	      "tags": ["mykey"],
        "settingIds": ["ID_FROM_STEP_ONE"],
        "rateLimitOverTime": 2,
        "rateLimitUnit": "m",
        "costLimitInUsd": 0.25
      }'   
```

### Congratulations you are done!!!
Then, just redirect your requests to us and use vLLM as you would normally. For example:
```bash
curl -X POST http://localhost:8002/api/providers/vllm/v1/chat/completions \
   -H "Authorization: Bearer my-vllm-key" \
   -H "Content-Type: application/json" \
   -d '{
          "model": "facebook/opt-125m",
          "messages": [
              {
                  "role": "system",
                  "content": "Where is San Francisco?"
              }
          ]
      }'
```

Or if you're using an SDK, you could change its `baseURL` to point to us. For example:
```js
// OpenAI Node SDK v4
import OpenAI from 'openai';

const openai = new OpenAI({
  apiKey: "my-vllm-key", // key created earlier
  baseURL: "http://localhost:8002/api/providers/vllm/v1", // redirect to us
});
```

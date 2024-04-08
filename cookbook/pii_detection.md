# PII filtering Guide

## Getting Started
This guide shows you how to create API keys that blocks requests that contains certain PIIs. Let's say you want to block http requests to OpenAI that contains `name` and `address`. Since the PII detection feature is powered by AWS's ML features, `AWS_SECRET_ACCESS_KEY` and `AWS_ACCESS_KEY_ID` are needed before starting BricksLLM.

### Step 0 - Set up AWS credentials as env variables
```bash
export AWS_SECRET_ACCESS_KEY=YOUR_AWS_SECRET_ACCESS_KEY
export AWS_ACCESS_KEY_ID=YOUR_AWS_ACCESS_KEY_ID
```
If you are running bricks using docker, you should update the env variables to include
```yaml
    environment:
      - AWS_SECRET_ACCESS_KEY=YOUR_AWS_SECRET_ACCESS_KEY
      - AWS_ACCESS_KEY_ID=YOUR_AWS_ACCESS_KEY_ID
```

### Step 1 - Start BricksLLM

Start BricksLLM.

### Step 2 - Create a provider
```bash
curl -X PUT http://localhost:8001/api/provider-settings \
   -H "Content-Type: application/json" \
   -d '{
          "provider":"openai",
          "setting": {
             "apikey": "YOUR_OPENAI_API_KEY"
          }
      }'   
```
Copy the `id` from the response.


### Step 3 - Create a policy
```bash
curl -X POST http://localhost:8001/api/policies \
   -H "Content-Type: application/json" \
   -d '{
          "name":"Name and address block policy",
          "tags": ["org-1"],
          "config": {
            "rules": {
                "name": "block",
                "address": "allow_but_redact"
            }
          }
      }'   
```
Copy the `id` from the response.

### Step 4 - Create a Bricks API key
```bash
curl -X PUT http://localhost:8001/api/key-management/keys \
   -H "Content-Type: application/json" \
   -d '{
	      "name": "My Secret Key",
	      "key": "my-secret-key",
	      "tags": ["org-1"],
          "settingIds": ["YOUR_SETTING_ID_FROM_STEP_TWO"],
          "policyId": "YOUR_POLICY_ID_FROM_STEP_THREE"
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

Trying to send requests with `name` or `address` will result in ```403```.

```bash
curl -X POST http://localhost:8002/api/providers/openai/v1/chat/completions \
   -H "Authorization: Bearer my-secret-key" \
   -H "Content-Type: application/json" \
   -d '{
          "model": "gpt-3.5-turbo",
          "messages": [
              {
                  "role": "user",
                  "content": "My name is Spike."
              }
          ]
      }'
```

This request will result in an `403`.

```bash
curl -X POST http://localhost:8002/api/providers/openai/v1/chat/completions \
   -H "Authorization: Bearer my-secret-key" \
   -H "Content-Type: application/json" \
   -d '{
          "model": "gpt-3.5-turbo",
          "messages": [
              {
                  "role": "user",
                  "content": "I live in 404 Hacker Way, San Francisco, CA."
              },
                {
                  "role": "system",
                  "content": "Where do I live?"
              }
          ]
      }'
```

This request will result in an `200`, but the LLM wouldn't know your address.

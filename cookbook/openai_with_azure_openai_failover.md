# OpenAI with Azure OpenAI as failover guide

## Getting Started
This guide shows you how to create a route that calls OpenAI chat completion with Azure as failover.

### Step 1 - Create an OpenAI provider
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

### Step 2 - Create an Azure OpenAI provider
```bash
curl -X PUT http://localhost:8001/api/provider-settings \
   -H "Content-Type: application/json" \
   -d '{
          "provider":"azure",
          "setting": {
                "resourceName": "YOUR_AZURE_RESOURCE_NAME",
                "apikey": "YOUR_AZURE_API_KEY"
          }
      }'   
```
Copy the `id` from the response.

### Step 3 - Create a Bricks API key
Use `id` from the step 2 and step 1 as `settingIds` to create a key.
```bash
curl -X PUT http://localhost:8001/api/key-management/keys \
   -H "Content-Type: application/json" \
   -d '{
	      "name": "My Secret Key",
	      "key": "my-secret-key",
          "settingIds": ["ID_FROM_STEP_TWO", "ID_FROM_STEP_ONE"]
      }'   
```

### Step 4 - Create a route that calls OpenAI chat completion with Azure OpenAI chat completion as fallback
Use `id` from the step 3 as part of `keyIds` to create a route with a caching TTL of 5 seconds, 2 retries for the initial OpenAI requests with a timeout of 10 seconds and 2 retries for the azure openai failover with a timeout of 10 seconds.

```bash
curl -X POST http://localhost:8001/api/routes \
   -H "Content-Type: application/json" \
   -d '{
    "name": "test",
    "path": "/test/chat/completion",
    "cacheConfig": {
        "enabled": true,
        "ttl": "5s"
    },
    "steps": [
        {
            "provider": "openai",
            "retries": 2,
            "model": "gpt-4",
            "params": {},
            "timeout": "10s"
        },
        {
            "provider": "azure",
            "retries": 2,
            "model": "gpt-4",
            "params": {
                "deploymentId": "YOUR_AZURE_DEPLOYMENT_ID",
                "apiVersion": "YOUR_DESIRED_AZURE_API_VERSION"
            },
            "timeout": "10s"
        }
    ],
    "keyIds": ["KEY_ID_FROM_STEP_THREE"]
}'   
```

### Congratulations you are done!!!
Then, just redirect your requests to the route that you just configured. For example:
```bash
curl -X POST http://localhost:8002/api/routes/test/chat/completion \
   -H "Authorization: Bearer my-secret-key" \
   -H "Content-Type: application/json" \
   -d '{
          "messages": [
              {
                  "role": "system",
                  "content": "hi"
              }
          ]
      }'
```

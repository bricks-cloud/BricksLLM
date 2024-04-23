# Access Control Based On User ID

## Use cases
* You have an internal AI powered application and would like to set model access based on people's email.
* You have an AI powered SaaS application and would like to enforce usage limits on for users on different tiers.

## Getting Started
This guide shows you how to do user level access control. Let's say you want to offer different spend limits, rate limits and model access for different users. 

### Step 1 - Create a provider
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

### Step 2 - Create a Bricks API key
Use `id` from the previous step as `settingId` to create a key.

```bash
curl -X PUT http://localhost:8001/api/key-management/keys \
   -H "Content-Type: application/json" \
   -d '{
	      "name": "My Secret Key",
	      "key": "my-secret-key",
	      "tags": ["team-one"],
          "settingIds": ["ID_FROM_STEP_ONE"]
      }'   
```

### Step 3 - Create a User
Use `tags` from the previous step and an `userId` that you defined to create a user.

```bash
curl -X POST http://localhost:8001/api/users \
   -H "Content-Type: application/json" \
   -d '{
            "name": "Spike Lu",
            "costLimitInUsd": 1,
            "costLimitInUsdOverTime": 0.002,
            "costLimitInUsdUnit": "m",
            "rateLimitOverTime": 5,
            "rateLimitUnit": "m",
            "allowedPaths": [
                {
                "path": "/api/providers/openai/v1/chat/completions",
                "method": "POST"
                }
            ],
            "allowedModels": ["gpt-4"],
            "userId": "my-user-id",
            "tags": ["team-one"]
      }'   
```

You just created a user referenced by the user id `my-user-id`. This user has a total spend limit of $1, a per minunte spend limit of $0.002, a rate limit of 5 req/m and can only access OpenAI's chat completion endpoint using `gpt-4`.

### Congratulations you are done!!!
Then, just redirect your requests to us and use OpenAI as you would normally with the same `userId` from step 3. For example:
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
          ],
          "user": "my-user-id"
    }'
```

This request would result in a `401`, since this user only has access to `gpt-4`.

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
          ],
          "user": "my-user-id"
    }'
```

This request would pass since this user is allowed to access `gpt-4`.
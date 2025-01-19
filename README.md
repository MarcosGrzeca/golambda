## Overview


# Build
docker build -t go-lambda-sqs .

# Run image
docker run -p 9000:8080 go-lambda-sqs



#Payload 

```
curl -X POST "http://localhost:9000/2015-03-31/functions/function/invocations" -d '{
    "Records": [
        {
            "messageId": "1",
            "attributes": {
                "MessageGroupID" : "10"
            },
            "body": "{\"url\": \"https://webhook.site/d04350dd-8b1c-4d71-a96e-6d51ba286b6d\", \"payload\": {\"product\": 10, \"sku\": 25}}",
            "Attempts": 10
        },
        {
            "messageId": "2",
            "attributes": {
                "MessageGroupID" : "11"
            },
            "body": "{\"url\": \"https://webhook.site/d04350dd-8b1c-4d71-a96e-6d51ba286b6d\", \"payload\": {\"product\": 10, \"sku\": 20}}",
            "Attempts": 10
        },
        {
            "messageId": "3",
            "attributes": {
                "MessageGroupID" : "11"
            },
            "body": "{\"url\": \"https://webhook.site/d04350dd-8b1c-4d71-a96e-6d51ba286b6d\", \"payload\": {\"product\": 10, \"sku\": 20}}",
            "Attempts": 10
        }
    ]
}'
```

# Tests
go test -v
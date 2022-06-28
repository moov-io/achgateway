#### Email

Example:

```
Sharding:
  Shards:
  - id: "production"
    notifications:
      email:
      - id: "production"
        from: "noreply@company.net"
        to:
          - "ach@bank.com"
        companyName: "Acme Corp"
```

#### PagerDuty

Example:

```
Sharding:
  Shards:
  - id: "production"
    notifications:
      pagerduty:
      - id: "production"
        apiKey: "..."
        from: "..."
        serviceKey: "..."
```

#### Slack

Example:

```
Sharding:
  Shards:
  - id: "production"
    notifications:
      slack:
      - id: "production"
        webhookURL: "https://hooks.slack.com/services/..."
```

## Shard Mappings

Shard mapping endpoints are exposed for persisting shard mappings to and retrieving shard mappings from the database, which map a shard key to a configured shard name

#### `POST /shard_mappings`

```
POST /shard_mappings
```

Create shard mappings

**Example Request Body / Payload**

```json
{
  "shard_key": "53ce45d6-aa44-4da8-8ebb-b3daf8c1886d",
  "ach_company_id": "testing"
}
```

**Response Codes:**
- 201 - Created - The request to create a resource was successful
- 400 - Bad Request - The request payload was not serializable
- 500 - Internal Server error - Unexpected error in server, no response other than error code

#### `GET /shard_mappings`

Get shard mappings list

**Example Response Body**

```json
[
  {
    "shard_key": "53ce45d6-aa44-4da8-8ebb-b3daf8c1886d",
    "shard_name": "testing"
  },
  {
    "shard_key": "55f177da-c389-42b1-87a2-5d6a14685690",
    "shard_name": "live"
  }
]
```

**Response Codes:**
- 200 - Success - The request to get a resource was successful
- 500 - Internal Server error - Unexpected error in server, no response other than error code

### `GET /shard_mappings/{shardKey}`

Get shard by shard key

**Example Response Body**

```json
{
  "shard_key": "53ce45d6-aa44-4da8-8ebb-b3daf8c1886d",
  "shard_name": "testing"
}
```

**Response Codes:**
- 200 - Success - The request to get a resource was successful
- 404 - Not Found - The requested resource was not found
- 500 - Internal Server error - Unexpected error in server, no response other than error code

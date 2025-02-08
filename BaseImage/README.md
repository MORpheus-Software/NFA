# BaseImage Proxy

A proxy service that integrates with the Lumerin Node API for model inference and session management.

## Features

- Bearer token authentication support
- Automatic session management with configurable expiration
- Model validation with similarity matching
- Streaming chat completions
- Health check endpoint
- Docker support with multi-arch builds

## Prerequisites

- Go 1.22 or later
- Docker and Docker Compose
- Access to a Lumerin Node API instance

## Configuration

Copy the example environment file and update it with your settings:

```bash
cp .env.example .env
```

Required environment variables:

- `LUMERIN_NODE_API`: URL of the Lumerin Node API (default: http://lumerin-node-api:8083)
- `AUTH_TOKEN`: Your authentication token for the Lumerin Node API
- `WALLET_ADDRESS`: Your Ethereum wallet address
- `WALLET_PRIVATE_KEY`: Your wallet's private key
- `MODEL_ID`: The ID of the model you want to use

Optional environment variables:

- `PORT`: Server port (default: 8080)
- `SESSION_DURATION`: Duration for session validity (default: 1h)
- `SESSION_EXPIRATION_SECONDS`: Session expiration in seconds (default: 1800)
- `LOG_LEVEL`: Logging level (default: info)

## Building and Running

### Local Development

```bash
# Build the binary
go build -o bin/nfa-proxy

# Run the proxy
./bin/nfa-proxy
```

### Docker

```bash
# Build the Docker image
docker compose build

# Start the service
docker compose up -d

# View logs
docker compose logs -f
```

## API Endpoints

### Health Check
```
GET /health
```

### Chat Completions
```
POST /v1/chat/completions
Authorization: Bearer <session_token>

{
  "model": "model-name",
  "messages": [
    {"role": "user", "content": "Hello"}
  ],
  "stream": true
}
```

### Get Available Models
```
GET /blockchain/models
```

## Testing

```bash
# Run all tests
go test -v ./...

# Run specific package tests
go test -v ./proxy/...
```

## Troubleshooting

1. If you see "no such host" errors, ensure the Lumerin Node API is accessible and properly configured in your environment.
2. For authentication errors, verify your AUTH_TOKEN is correctly set and valid.
3. For model validation errors, ensure the model name matches closely with available models (90% similarity required).

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a new Pull Request
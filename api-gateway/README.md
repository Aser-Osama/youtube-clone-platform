# API Gateway

This is the API Gateway for the YouTube-style video streaming platform. It routes and secures requests to backend services, handles authentication, rate limiting, and circuit breaking, and proxies OAuth flows.

## Features

- JWT validation (RS256, public key)
- Google OAuth login via reverse proxy to auth-service
- Rate limiting per IP
- Circuit breaker for backend service resilience
- Transparent proxying of cookies and redirects for OAuth
- Clean architecture, modular codebase

## Configuration

- All configuration is via environment variables in a `.env` file (see `scripts/setup.sh` for an example).
- Example `.env`:
  ```env
  SERVER_PORT=8085
  AUTH_SERVICE_URL=http://localhost:8080
  METADATA_SERVICE_URL=http://localhost:8082
  STREAMING_SERVICE_URL=http://localhost:8090
  UPLOAD_SERVICE_URL=http://localhost:8081
  TRANSCODER_SERVICE_URL=http://localhost:8083
  JWT_PUBLIC_KEY_PATH=keys/public.pem
  RATE_LIMIT_REQUESTS=100
  RATE_LIMIT_PERIOD=1m
  ```

## Running

1. Ensure all dependencies and backend services are running.
2. Generate `.env` with `./scripts/setup.sh` (edit as needed).
3. Download dependencies:
   ```sh
   go mod tidy
   ```
4. Start the gateway:
   ```sh
   go run cmd/main.go
   ```

## Testing

- See [TESTING.md](./TESTING.md) for detailed instructions.
- Run automated tests:
  ```sh
  chmod +x scripts/test_gateway.sh scripts/test_with_token.sh
  ./scripts/test_gateway.sh
  ./scripts/test_with_token.sh <jwt_token>
  ```
- For browser-based OAuth flow, open `http://localhost:8085/auth/login` in your browser and follow the prompts.

## Project Structure

- `cmd/` - Main entry point
- `internal/` - Core logic (config, service, middleware)
- `scripts/` - Test and setup scripts

## Notes

- The gateway is designed to be stateless and scalable.
- For production, ensure secure cookie settings and HTTPS termination at the load balancer or gateway.

---

For more details, see the main project README and service-specific documentation.

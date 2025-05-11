# Testing the API Gateway

This document describes how to test the API Gateway, including browser-based OAuth flows and API endpoint checks.

## 1. Prerequisites

- All backend services (`auth-service`, `api-gateway`, etc.) are running.
- `.env` is configured and present in the `api-gateway` directory.
- The gateway is running on `http://localhost:8085`.

## 2. Browser-Based OAuth Flow

### A. Initiate Login

1. Open your browser.
2. Navigate to: `http://localhost:8085/auth/login`
3. You should be redirected to Google's OAuth consent screen.

### B. Complete Google Login

4. Log in with your Google account and approve the app.

### C. Callback Handling

5. After login, you will be redirected to `/auth/callback` on the gateway.
6. The gateway proxies this to the `auth-service`, which should:
   - Set authentication cookies (e.g., refresh token).
   - Redirect you to your frontend (e.g., `http://localhost:3000/`).

### D. Verify Cookies

7. Open browser dev tools and check cookies for your domain. You should see tokens set by the `auth-service`.

### E. Troubleshooting

- If you see errors or missing cookies, check the browser's network tab for failed requests, missing cookies, or CORS issues.

## 3. API Endpoint Testing (Manual/Curl)

### A. Health Check

```sh
curl http://localhost:8085/health
```

Should return `{ "status": "ok" }`.

### B. Rate Limiting

Send more requests than allowed in the configured period:

```sh
for i in {1..101}; do curl -s http://localhost:8085/health; done
```

You should see a rate limit error after the threshold.

### C. JWT-Protected Endpoints

- Try accessing `/auth/refresh`, `/metadata/videos`, etc. without a token:

```sh
curl http://localhost:8085/auth/refresh
```

Should return an error about missing authorization.

- Try with an invalid token:

```sh
curl -H "Authorization: Bearer invalid.token.here" http://localhost:8085/auth/refresh
```

Should return an error about invalid token.

### D. Auth Proxy Endpoints

- Test `/auth/login` and `/auth/callback` in the browser as above.
- Test `/auth/refresh` and `/auth/logout` with cookies (browser) or curl (with `-b`/`-c` for cookies).

### E. OPTIONS Preflight

```sh
curl -X OPTIONS http://localhost:8085/metadata/videos
```

Should return an empty or allowed response.

## 4. Automated Test Scripts

Run the provided test scripts:

```sh
chmod +x scripts/test_gateway.sh scripts/test_with_token.sh
./scripts/test_gateway.sh
./scripts/test_with_token.sh <jwt_token>
```

## 5. Notes

- For full OAuth flow, use a browser.
- For API endpoint checks, use curl or the provided scripts.
- If you encounter issues, check logs for both the gateway and the `auth-service`.

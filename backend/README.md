# GlowMeet backend (X login)

Golang chi demo that handles X.com OAuth login with PKCE.

## Setup

1) Copy env: `cp .env.example .env` and fill `X_CLIENT_ID`, `X_CLIENT_SECRET`, `X_REDIRECT_URL` (must match your X app settings).  
2) Run: `go run main.go` from the `backend` directory.  
3) Backend defaults to `:8080` and allows CORS from `CORS_ORIGIN`.

## Endpoints

- `GET /health` — readiness probe.  
- `GET /auth/x/login` — returns `authorization_url` and `state` you can redirect the user to.  
- `GET /auth/x/callback?code=...&state=...` — exchanges the code using the stored PKCE verifier; responds with access/refresh tokens.

State + PKCE verifiers live in-memory with a 10 minute TTL; wire your own session or persistence layer for production.

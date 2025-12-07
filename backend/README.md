# GlowMeet backend (X login)

Golang chi demo that handles X.com OAuth login with PKCE.

## Setup

1) Copy env: `cp .env.example .env` and fill `X_CLIENT_ID`, `X_CLIENT_SECRET`, `X_REDIRECT_URL` (match your X app redirect; use the frontend origin like `http://localhost:3000/auth/x/callback` when proxying), and `APP_JWT_SECRET`. `FRONTEND_URL` can be a relative path (default `/`) to avoid hardcoded localhost redirects.  
2) Run: `go run main.go` from the `backend` directory.  
3) Backend defaults to `:8000` and allows CORS from `CORS_ORIGIN`.

## Endpoints

- `GET /health` — readiness probe.  
- `GET /auth/x/login` — returns `authorization_url` and `state` you can redirect the user to.  
- `GET /auth/x/callback?code=...&state=...` — exchanges the code using the stored PKCE verifier; creates a JWT app session cookie `access_token` (sub = X user id), stores the X OAuth token server-side keyed by user id, and redirects to `FRONTEND_URL`.  
- `GET /api/me` — uses the session cookie to look up the stored X token and fetch the user profile from X.com.  
- `POST /api/me` — updates the user's geolocation. Expects JSON struct: `{"lat": 37.7749, "long": -122.4194}`.
- `GET /api/users` — returns up to 20 recently seen users (in-memory list).

State + PKCE verifiers + user list live in-memory; wire your own session or persistence layer for production.

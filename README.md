# OneLab-API

A small Go API gateway ("OneAPI") that unifies several self-hosted services behind a single
authenticated HTTP interface. Integrations are configurable and the live set, their status, and
their settings are reported by the running app.

---

## Requirements

- [Go 1.25+](https://go.dev/dl/) (to build/run locally)
- [Docker](https://docs.docker.com/get-docker/) (optional, for containerized runs)

Each integration is optional: the API starts without them and reports per-integration status at
`/api/v1/status`.

---

## Setup

```bash
git clone https://github.com/Henriii-01/OneLab-API.git
cd OneLab-API
go mod download
cp .env.example .env   # fill in your values
```

Configuration comes from two places:

- **Environment variables** — secrets and per-integration connection details. See
  [`.env.example`](.env.example) for the full, current list. The app reads these directly from the
  process environment.
- **`config/config.json`** — non-secret settings (e.g. token expiry). Sensible defaults apply when
  the file or a value is missing.

---

## API

All routes are served under `/api/v1` (except `/health`). Most routes require a bearer token —
obtain one from the configured client credentials:

```bash
curl -X POST http://localhost:8080/api/v1/auth/token \
  -H "Content-Type: application/json" \
  -d '{"clientId": "<id>", "clientSecret": "<secret>"}'
# -> { "token": "<jwt>", "expiresIn": <seconds> }
```

Send it on protected routes via `Authorization: Bearer <token>`.

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `GET` | `/health` | none | Liveness probe |
| `GET` | `/api/v1/status` | none | Per-integration connectivity |
| `POST` | `/api/v1/auth/token` | none | Issue a signed token from client credentials |

Integration-specific routes (file transfer, lookup, etc.) are added per integration. Inspect the
registered controllers in the source for the current set.

# Minutely API

Go backend for Minutely authentication and health-check endpoints.

## Requirements

- Go `1.26.1` as declared in [`go.mod`](/home/dexter/Minutely/minutely-api/go.mod)
- A Supabase project

## Setup

1. Clone the repository and enter it:

```bash
git clone <repo-url>
cd minutely-api
```

2. Create a `.env` file in the project root:

```env
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_KEY=your-supabase-key
```

The app loads `.env` on startup and exits if either variable is missing.

## Install Dependencies

Go will download dependencies automatically the first time you run the app:

```bash
go mod download
```

## Run The Project

Start the API server with:

```bash
go run ./cmd/api
```

The server starts on:

```text
http://127.0.0.1:8080
```

## Verify It Is Running

Health check:

```bash
curl http://127.0.0.1:8080/api/health
```

Expected response:

```json
{"status":"success","message":"Minutely Go backend is fully operational!"}
```

Example signup request:

```bash
curl -X POST http://127.0.0.1:8080/api/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secret123"}'
```

Example login request:

```bash
curl -X POST http://127.0.0.1:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secret123"}'
```

## Checks

Dependencies and package compilation were verified with:

```bash
go test ./...
```

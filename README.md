# Minutely API

The Go-based backend server for the Minutely application suite. It provides REST API endpoints for authentication, user profiles, and meeting management.

## Setup

Ensure you have Go installed on your system.

```bash
cd minutely-api
go mod download
```

## How to Run

To run the server locally:

```bash
go run cmd/api/main.go
```

The server will start and listen for incoming HTTP requests from the `minutely-desktop` and `minutely-web` clients. Ensure your frontend clients are configured to point to the correct local address.

## Confirmed Routes

The following routes are currently confirmed to be active:
- `/api/health`
- `/api/auth/signup`, `/api/auth/login`, `/api/auth/logout`
- `/api/user/profile`
- `/api/preferences/theme`
- `/api/meetings/*` (next, recent, schedule, instant, validate, end, participants)
- `/api/webrtc/*` (signal, signals)

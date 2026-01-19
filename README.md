# Sessions Library

Go library for JWT-based session management with optional Redis-backed persistence. Provides a simple interface to create, validate, refresh, and delete session tokens, with sorted-set storage for revocation/expiration checks.

## Features
- Stateless JWT tokens (HS256) with configurable expiration
- Optional Redis persistence for token revocation and multi-session tracking
- In-memory storage for fast tests and local usage
- Context-aware APIs for cancellation/timeouts
- Error wrapping with Go 1.13+ `%w`
- Makefile helpers for tidy, update, and tests

## Requirements
- Go 1.24+
- (Optional) Redis 6+ when using Redis storage

## Installation
```bash
go get github.com/andskur/sessions
```

## Quick Start (JWT only)
```go
ctx := context.Background()
secret := []byte("your-secret-key")
expire := 24 * time.Hour
sessions, err := jwt.NewJwtSession(secret, expire, nil)
if err != nil {
    log.Fatal(err)
}

data := map[string]interface{}{
    "user_id": 123,
    "email": "user@example.com",
}
token, err := sessions.Create(ctx, data)
if err != nil {
    log.Fatal(err)
}

info, err := sessions.Get(ctx, token)
if err != nil {
    log.Fatal(err)
}
fmt.Println(info)
```

## With Redis Persistence
```go
ctx := context.Background()
storage := redis.New(&goredis.Options{Addr: "localhost:6379", DB: 0})
if _, err := storage.Ping(ctx); err != nil {
    log.Fatal(err)
}

secret := []byte("your-secret-key")
expire := 24 * time.Hour
sessions, err := jwt.NewJwtSession(secret, expire, storage)
if err != nil {
    log.Fatal(err)
}

userID := uuid.Must(uuid.NewV4())
data := map[string]interface{}{
    "uuid": userID,
    "role": "admin",
}
token, err := sessions.Create(ctx, data)
if err != nil {
    log.Fatal(err)
}

// Later: revoke
if err := sessions.Delete(ctx, token); err != nil {
    log.Fatal(err)
}
```

## In-Memory Storage (tests/local)
```go
ctx := context.Background()
storage := memory.New()
secret := []byte("secret")
sessions, _ := jwt.NewJwtSession(secret, time.Hour, storage)
```

## API Overview
- `Create(ctx, data)` → `Token`
- `Get(ctx, token)` → `map[string]interface{}`
- `RefreshToken(ctx, oldToken)` → new `Token`
- `Delete(ctx, token)` → revoke (requires storage)

### Data requirements when using storage
- `uuid` field (`uuid.UUID`) must be present in `data` for storage-backed sessions.

### Special `source` field
- If `data["source"]` exists, token expiration is set to `999999h` (long-lived service tokens).

## Makefile Targets
- `make tidy`   — run `go mod tidy`
- `make update` — run `go get -u ./...`
- `make tests`  — run `go test ./...`

## Running Tests
```bash
make tests
```

## Error Semantics
- `ErrUnexpectedToken` — invalid token format/signature
- `ErrNotFound` — token not registered or already deleted
- `ErrExpired` — token expired
- `ErrNoStorage` — delete called without configured storage
- `ErrDataNotValid` — missing/invalid data (e.g., uuid required for storage)

## Notes & Recommendations
- Use strong secrets and HTTPS when transmitting tokens.
- Prefer Redis storage in production to enable revocation/blacklisting.
- Contexts allow cancellation/timeouts for all operations.

## Contributing
Contributions are welcome! Please feel free to submit a Pull Request.

For development guidelines and best practices, see `AGENTS.md`.

## License
This project is licensed under the MIT License — see the `LICENSE` file for details.

## Author
Copyright (c) 2022 Andrey Skurlatov

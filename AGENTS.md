# Sessions Library - Developer Guide

## Overview

This is a Go library that provides JWT-based session management with optional Redis persistence. It offers a clean interface for creating, validating, refreshing, and deleting user sessions with support for token blacklisting and expiration.

## Architecture

### Core Components

```
sessions/
├── sessions.go          # Core ISessions interface
├── errors.go            # Session-related errors
├── jwt/
│   └── jwt.go          # JWT implementation of ISessions
└── persistance/
    ├── nosql.go        # Storage interface definitions
    ├── errors.go       # Storage errors
    └── redis/
        ├── redis.go    # Redis storage implementation
        ├── set.go      # Redis sorted set wrapper
        └── errors.go   # Redis error handling
```

### Design Patterns

1. **Interface-based design**: Core functionality defined through `ISessions` interface
2. **Storage abstraction**: `IStorage` interface allows multiple backend implementations
3. **Error wrapping**: Uses Go 1.13+ error wrapping (`fmt.Errorf` with `%w`)
4. **Sorted sets for expiration**: Redis ZSET with Unix timestamps as scores for automatic cleanup

## Main Interfaces

### ISessions Interface

Location: `sessions.go:8-21`

The main interface for session management:

```go
type ISessions interface {
    // Create creates new session with associated data
    Create(ctx context.Context, data map[string]interface{}) (Token, error)
    
    // Get returns data associated with this token
    Get(ctx context.Context, token Token) (data map[string]interface{}, err error)
    
    // RefreshToken by creating and return new one
    RefreshToken(ctx context.Context, oldToken Token) (Token, error)
    
    // Delete makes token invalid
    Delete(ctx context.Context, token Token) error
}
```

### IStorage Interface

Location: `persistance/nosql.go:9-40`

Abstraction for NoSQL key-value storage:

```go
type IStorage interface {
    Ping(ctx context.Context) (string, error)
    Get(ctx context.Context, key string) (data interface{}, err error)
    Set(ctx context.Context, key string, data interface{}) error
    SetWithExpire(ctx context.Context, key string, data interface{}, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    CountKeys(ctx context.Context, pattern string) (count int, err error)
    StrSet(key string) IStrSet
    Close() error
}
```

### IStrSet Interface

Location: `persistance/nosql.go:44-59`

Set operations with expiration support:

```go
type IStrSet interface {
    Add(ctx context.Context, val string) error
    AddExpire(ctx context.Context, val string, ttl time.Duration) error
    Remove(ctx context.Context, val string) error
    Check(ctx context.Context, val string) (bool, error)
    List(ctx context.Context) ([]string, error)
}
```

## JWT Implementation Details

### Token Structure

Location: `jwt/jwt.go:14-19`

JWT tokens contain:
- **Standard claims**:
  - `exp` (expireAtKey) - Expiration timestamp
  - `iat` (issuedAtKey) - Issued at timestamp
- **Custom claims**:
  - `persistKey` - Token ID stored in Redis (if storage enabled)
  - `uuid` - User UUID
  - Any additional data passed to `Create()`

### Token Creation Flow

Location: `jwt/jwt.go:47-88`

1. Creates new JWT with HS256 signing method
2. Copies user data into claims
3. Sets `iat` (issued at) to current time
4. Sets expiration:
   - If `source` field exists: 999999 hours (special case)
   - Otherwise: uses configured expire duration
5. If storage enabled:
   - Generates UUID for token ID
   - Stores token ID in Redis sorted set: `user:{uuid}:sessions`
   - Uses expiration time as ZSET score
6. Signs and returns token

### Token Validation Flow

Location: `jwt/jwt.go:93-149`

1. Extracts and validates JWT claims
2. Checks signature using secret key
3. Verifies expiration timestamp
4. If storage enabled:
   - Extracts token ID from claims
   - Checks Redis for token ID existence
   - Returns `ErrNotFound` if not in storage
5. Returns session data (excluding technical fields)

### Token Refresh Flow

Location: `jwt/jwt.go:152-163`

1. Validates old token via `Get()`
2. Deletes old token via `Delete()`
3. Creates new token with same data via `Create()`

### Token Deletion Flow

Location: `jwt/jwt.go:166-192`

1. Requires storage to be configured (returns `ErrNoStorage` otherwise)
2. Extracts token ID from claims
3. Removes token ID from Redis sorted set
4. Returns `ErrNotFound` if token wasn't in storage

## Redis Implementation Details

### Storage Key Format

Location: `jwt/jwt.go:234-236`

User sessions stored as: `user:{uuid}:sessions`

### Sorted Set Strategy

Location: `persistance/redis/set.go`

Uses Redis ZSET (sorted sets) where:
- **Member**: Token ID (UUID string)
- **Score**: Unix timestamp of expiration
- **Benefits**:
  - Automatic ordering by expiration time
  - Efficient range queries
  - Built-in cleanup with `ZREMRANGEBYSCORE`

### Expiration Handling

Location: `persistance/redis/set.go:23-41`

On every `AddExpire()`:
1. Removes expired tokens: `ZREMRANGEBYSCORE key -inf {now-1}`
2. Adds new token with expiration as score
3. Special handling: `-1` TTL = infinite (math.Inf)

### Validation Check

Location: `persistance/redis/set.go:50-57`

`Check()` verifies token is both:
1. Present in sorted set (`ZSCORE`)
2. Not expired (score >= current Unix timestamp)

## Error Handling

### Session Errors

Location: `errors.go:8-14`

```go
ErrUnexpectedToken - Invalid token format
ErrNotFound        - Token not registered or deleted
ErrExpired         - Token expired
ErrNoStorage       - No storage configured (for Delete)
ErrDataNotValid    - Invalid data provided
```

### Storage Errors

Location: `persistance/errors.go:9-12`

```go
ErrNoSuchKeyFound - Key doesn't exist
ErrNotStrSet      - Value is not a string set
```

### Error Wrapping

All errors are wrapped with context using `fmt.Errorf` with `%w` verb for proper error chain inspection.

## Usage Examples

### Basic Setup (JWT only, no persistence)

```go
import (
    "context"
    "time"
    "github.com/andskur/sessions/jwt"
)

ctx := context.Background()
secret := []byte("your-secret-key")
expire := 24 * time.Hour
sessions, err := jwt.NewJwtSession(secret, expire, nil)
if err != nil {
    panic(err)
}

data := map[string]interface{}{
    "user_id": 123,
    "email": "user@example.com",
}
token, err := sessions.Create(ctx, data)
if err != nil {
    panic(err)
}

sessionData, err := sessions.Get(ctx, token)
if err != nil {
    panic(err)
}
```

### With Redis Persistence

```go
import (
    "context"
    "time"
    "github.com/andskur/sessions/jwt"
    "github.com/andskur/sessions/persistance/redis"
    goredis "github.com/redis/go-redis/v9"
    "github.com/gofrs/uuid"
)

ctx := context.Background()
storage := redis.New(&goredis.Options{Addr: "localhost:6379", DB: 0})
if _, err := storage.Ping(ctx); err != nil {
    panic(err)
}

secret := []byte("your-secret-key")
expire := 24 * time.Hour
sessions, err := jwt.NewJwtSession(secret, expire, storage)
if err != nil {
    panic(err)
}

userID := uuid.Must(uuid.NewV4())
data := map[string]interface{}{
    "uuid": userID,
    "user_id": 123,
    "email": "user@example.com",
}
token, err := sessions.Create(ctx, data)

err = sessions.Delete(ctx, token)
```

### Refresh Token

```go
ctx := context.Background()
newToken, err := sessions.RefreshToken(ctx, oldToken)
if err != nil {
    panic(err)
}
```

## Important Notes

### Data Requirements

When using Redis storage, the data map **must** contain:
- `uuid` field with type `uuid.UUID`

### Special "source" Field

If `data` contains a `source` field:
- Token expiration set to 999999 hours (~114 years)
- Appears to be a special case for service tokens

### Type Assertions

Code contains type assertions:
```go
claims["uuid"].(string)
```
Ensure valid types to avoid panics.

### Security Considerations

1. **Secret Key**: Must be strong and kept secure
2. **HTTPS Only**: JWT tokens should only be transmitted over HTTPS
3. **Storage**: Redis persistence enables token revocation/blacklisting
4. **Expiration**: Always set reasonable expiration times

### Deprecated Dependencies (fixed)

- Replaced `dgrijalva/jwt-go` with `golang-jwt/jwt/v5`
- Upgraded `go-redis/redis` to v9

## Testing Strategy

The repository now includes context-aware interfaces and an in-memory storage implementation for fast tests.
Current/Planned unit tests cover:
- JWT Create/Get/Refresh/Delete flows
- Expiration handling
- Storage interactions (using in-memory storage; Redis tests can use miniredis)
- Error cases and edge conditions
- Context cancellation behavior

### Makefile targets
- `make tidy`   — run `go mod tidy`
- `make update` — run `go get -u ./...`
- `make tests`  — run `go test ./...`

Recommended test command:
```bash
make tests
```

### Documentation
See `README.md` for quick start, examples, and API overview.

## Contributing Guidelines

1. Follow standard Go formatting (`gofmt`)
2. Use error wrapping with `%w`
3. Add godoc comments for exported functions
4. Return errors, never panic

## Performance Considerations

- Redis operations are O(log N) for add/remove, O(1) for lookup
- JWT signing/verification is O(1)

## Roadmap & Future Improvements

- Add comprehensive test suite (in progress)
- Add usage examples in README
- Document "source" field special case or remove it
- Add context support (done)
- Update dependencies (done)

## References

- [JWT RFC 7519](https://www.rfc-editor.org/rfc/rfc7519)
- [Redis Sorted Sets](https://redis.io/docs/data-types/sorted-sets/)
- [Go Error Handling](https://go.dev/blog/go1.13-errors)

**Last Updated**: January 2026  
**Go Version**: 1.24  
**Maintainer**: Andrey Skurlatov

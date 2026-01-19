package sessions

import "context"

// Token represents user session token
type Token []byte

// ISessions collects, persist and manages user
// auth sessions via tokens and associated data
type ISessions interface {
	// Create creates new session
	Create(ctx context.Context, data map[string]interface{}) (Token, error)

	// Get returns data associated with this token
	Get(ctx context.Context, token Token) (data map[string]interface{}, err error)

	// RefreshToken by creating and return new one
	RefreshToken(ctx context.Context, oldToken Token) (Token, error)

	// Delete makes token invalid so sequential
	// Get call will returns ErrNotFound
	Delete(ctx context.Context, token Token) error
}

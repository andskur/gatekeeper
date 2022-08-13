package sessions

// Token represents user session token
type Token []byte

// ISessions collects, persist and manages user
// auth sessions via tokens and associated data
type ISessions interface {
	// Create creates new session
	Create(data map[string]interface{}) (Token, error)

	// Get returns data associated with this token
	Get(token Token) (data map[string]interface{}, err error)

	// RefreshToken by creating and return new one
	RefreshToken(oldToken Token) (Token, error)

	// Delete makes token invalid so sequential
	// Get call will returns ErrNotFound
	Delete(token Token) error
}

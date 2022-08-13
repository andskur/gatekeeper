package jwt

import (
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gofrs/uuid"

	"github.com/andskur/sessions"
	nosql "github.com/andskur/sessions/persistance"
)

const (
	tokenPersisIDKey = "persistKey"
	expireAtKey      = "exp"
	issuedAtKey      = "iat"
	signingMethod    = "HS256"
)

// SessionJwt implements storage interface using jwt-token mechanism
// where all optional data stored on the user side.
type SessionJwt struct {
	signingMethod jwt.SigningMethod
	secretKey     []byte
	expire        time.Duration
	storage       nosql.IStorage
}

// NewJwtSession creates new jwt storage
func NewJwtSession(secret []byte, expire time.Duration, storage nosql.IStorage) (sessions.ISessions, error) {
	Jwt := &SessionJwt{
		secretKey:     secret,
		expire:        expire,
		signingMethod: jwt.GetSigningMethod(signingMethod),
	}

	if storage != nil {
		Jwt.storage = storage
	}

	return Jwt, nil
}

// Create generates new jwt token
// and return it as a signed string
func (j *SessionJwt) Create(data map[string]interface{}) (sessions.Token, error) {
	token := jwt.New(j.signingMethod)
	claims := token.Claims.(jwt.MapClaims)

	// copy data
	for key, val := range data {
		claims[key] = val
	}

	// populate
	claims[issuedAtKey] = time.Now().Unix()

	if _, ok := data["source"]; ok {
		claims[expireAtKey] = time.Now().Add(999999 * time.Hour).Unix()
	} else {
		claims[expireAtKey] = time.Now().Add(j.expire).Unix()
	}

	// if token storage is defined, use it to store token session
	if j.storage != nil {
		tokenID, err := uuid.NewV4()
		if err != nil {
			return nil, fmt.Errorf("create token UUID: %w", err)
		}
		claims[tokenPersisIDKey] = tokenID.String()

		userID, ok := data["uuid"]
		if !ok {
			return nil, fmt.Errorf("create token: %w - %s", sessions.ErrDataNotValid, "no such user UUID")
		}

		key := formatStorageKey(userID.(uuid.UUID).String())

		if err = j.storage.StrSet(key).AddExpire(tokenID.String(), j.expire); err != nil {
			return sessions.Token{}, fmt.Errorf("create token: %w", err)
		}
	}

	// generate token string
	tokenString, err := token.SignedString(j.secretKey)
	return sessions.Token(tokenString), err
}

// Get validates token, checks expiration,
// checks storage for such token if it present
// and return data form given token
func (j *SessionJwt) Get(token sessions.Token) (map[string]interface{}, error) {
	claims, err := j.extractClaims(token)
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	// check token expired
	{
		expireAtTs, err := extractTimestamp(claims, expireAtKey)
		if err != nil {
			return nil, fmt.Errorf("validate token: %w", err)
		}
		if time.Now().Unix() > expireAtTs {
			return nil, fmt.Errorf("validate token: %w", sessions.ErrExpired)
		}
	}

	// if token storage defined, check token
	{
		if j.storage != nil {
			tokenID, err := extractTokenID(claims)
			if err != nil {
				return nil, fmt.Errorf("check token: %w", err)
			}

			// lookup token
			key := formatStorageKey(claims["uuid"].(string))
			exists, err := j.storage.StrSet(key).Check(tokenID)
			if !exists || err == nosql.ErrNoSuchKeyFound {
				return nil, sessions.ErrNotFound
			}
			if err != nil {
				return nil, fmt.Errorf("lookup token: %w", err)
			}
		}
	}

	// copy payload values except technical one
	data := make(map[string]interface{}, len(claims))
	for key, val := range claims {
		if key == expireAtKey || key == issuedAtKey || key == tokenPersisIDKey {
			continue
		}

		switch key {
		case "uuid":
			data[key], err = uuid.FromString(val.(string))
			if err != nil {
				return nil, fmt.Errorf("format token data: %w", err)
			}
		default:
			data[key] = val
		}
	}

	return data, nil
}

// RefreshToken refresh given token
func (j *SessionJwt) RefreshToken(oldToken sessions.Token) (sessions.Token, error) {
	data, err := j.Get(oldToken)
	if err != nil {
		return sessions.Token{}, fmt.Errorf("refresh token: %w", err)
	}

	if err = j.Delete(oldToken); err != nil {
		return sessions.Token{}, fmt.Errorf("refresh token: %w", err)
	}

	return j.Create(data)
}

// Delete token associated id in storage if it present
func (j *SessionJwt) Delete(token sessions.Token) error {
	// if there is no session storage, jwt token can't be deleted
	if j.storage == nil {
		return fmt.Errorf("delete token: %w", sessions.ErrNoStorage)
	}

	// extract claims
	claims, err := j.extractClaims(token)
	if err != nil {
		return fmt.Errorf("delete token: %w", err)
	}

	// extract token id
	tokenID, err := extractTokenID(claims)
	if err != nil {
		return fmt.Errorf("delete token: %w", err)
	}

	// remove it from tokens storage
	key := formatStorageKey(claims["uuid"].(string))

	if err = j.storage.StrSet(key).Remove(tokenID); err == nosql.ErrNoSuchKeyFound {
		return fmt.Errorf("delete token: %w", sessions.ErrNotFound)
	}

	return err
}

// extractClaims extract token claims from session token
func (j *SessionJwt) extractClaims(token sessions.Token) (jwt.MapClaims, error) {
	decodedToken, err := jwt.Parse(string(token), func(token *jwt.Token) (interface{}, error) {
		if token.Method != j.signingMethod {
			return nil, fmt.Errorf("extract claims: %w: invalid signing method: %s", sessions.ErrUnexpectedToken, token.Method)
		}
		return j.secretKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("extract claims: %w", err)
	}

	return decodedToken.Claims.(jwt.MapClaims), err
}

// extractTimestamp extract token timestamp from its claims
func extractTimestamp(claims jwt.MapClaims, key string) (int64, error) {
	if expireAt, ok := claims[key]; ok {
		if expireAt, ok := expireAt.(float64); ok {
			return int64(expireAt), nil
		}
	}
	return 0, fmt.Errorf("extract timestamp: %w", sessions.ErrUnexpectedToken)
}

// extractTokenID extract token id from its claims
func extractTokenID(claims jwt.MapClaims) (tokenID string, err error) {
	// validate payload token id
	if tokenIDRaw, ok := claims[tokenPersisIDKey]; ok {
		tokenID, ok = tokenIDRaw.(string)
		if !ok {
			err = fmt.Errorf("exctract token ID: %w", sessions.ErrUnexpectedToken)
		}
	} else {
		err = fmt.Errorf("exctract token ID: %w", sessions.ErrUnexpectedToken)
	}
	return
}

// formatStorageKey create key for NoSql storage
func formatStorageKey(uuid string) string {
	return fmt.Sprintf("user:%v:sessions", uuid)
}

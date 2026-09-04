package services

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

type TokenIdentity struct {
	UserID   int
	ArtistID int
	Email    string
}

type JWTService struct {
	Secret []byte
}

func NewJWTService(secret string) *JWTService {
	return &JWTService{
		Secret: []byte(secret),
	}
}

// GenerateToken creates the legacy artist token.
// Keep temporarily while old artist authentication still exists.
func (j *JWTService) GenerateToken(
	artistID int,
	email string,
) (string, error) {
	now := time.Now()

	claims := jwt.MapClaims{
		"artist_id": artistID,
		"email":     email,
		"exp":       now.Add(24 * time.Hour).Unix(),
		"iat":       now.Unix(),
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	return token.SignedString(j.Secret)
}

// GenerateUserToken is the new authentication token.
// New authentication is based on users.id.
func (j *JWTService) GenerateUserToken(
	userID int,
	email string,
) (string, error) {
	now := time.Now()

	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"exp":     now.Add(24 * time.Hour).Unix(),
		"iat":     now.Unix(),
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	return token.SignedString(j.Secret)
}

func (j *JWTService) ValidateIdentity(
	tokenString string,
) (TokenIdentity, error) {
	token, err := jwt.Parse(
		tokenString,
		func(token *jwt.Token) (interface{}, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, ErrInvalidToken
			}

			return j.Secret, nil
		},
	)

	if err != nil || !token.Valid {
		return TokenIdentity{}, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return TokenIdentity{}, ErrInvalidToken
	}

	identity := TokenIdentity{}

	if value, exists := claims["user_id"]; exists {
		id, err := claimInt(value)
		if err != nil {
			return TokenIdentity{}, ErrInvalidToken
		}

		identity.UserID = id
	}

	if value, exists := claims["artist_id"]; exists {
		id, err := claimInt(value)
		if err != nil {
			return TokenIdentity{}, ErrInvalidToken
		}

		identity.ArtistID = id
	}

	if value, exists := claims["email"]; exists {
		email, ok := value.(string)
		if ok {
			identity.Email = email
		}
	}

	if identity.UserID <= 0 && identity.ArtistID <= 0 {
		return TokenIdentity{}, ErrInvalidToken
	}

	return identity, nil
}

// ValidateToken remains temporarily for compatibility with existing tests
// and code that still expects an artist token.
func (j *JWTService) ValidateToken(
	tokenString string,
) (int, error) {
	identity, err := j.ValidateIdentity(tokenString)
	if err != nil {
		return 0, err
	}

	if identity.ArtistID <= 0 {
		return 0, ErrInvalidToken
	}

	return identity.ArtistID, nil
}

func claimInt(value interface{}) (int, error) {
	var id int

	switch value := value.(type) {
	case float64:
		id = int(value)

	case int:
		id = value

	case string:
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return 0, ErrInvalidToken
		}

		id = parsed

	default:
		return 0, ErrInvalidToken
	}

	if id <= 0 {
		return 0, ErrInvalidToken
	}

	return id, nil
}

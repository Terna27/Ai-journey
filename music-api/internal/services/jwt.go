package services

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

type JWTService struct {
	Secret []byte
}

func NewJWTService(secret string) *JWTService {
	return &JWTService{
		Secret: []byte(secret),
	}
}

func (j *JWTService) GenerateToken(
	artistID int,
	email string,
) (string, error) {

	claims := jwt.MapClaims{
		"artist_id": artistID,
		"email":     email,
		"exp":       time.Now().Add(24 * time.Hour).Unix(),
		"iat":       time.Now().Unix(),
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	return token.SignedString(j.Secret)
}

func (j *JWTService) ValidateToken(tokenString string) (int, error) {

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
		return 0, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, ErrInvalidToken
	}

	artistIDValue, ok := claims["artist_id"]
	if !ok {
		return 0, ErrInvalidToken
	}

	var artistID int

	switch value := artistIDValue.(type) {
	case float64:
		artistID = int(value)

	case int:
		artistID = value

	case string:
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return 0, ErrInvalidToken
		}

		artistID = parsed

	default:
		return 0, ErrInvalidToken
	}

	if artistID <= 0 {
		return 0, ErrInvalidToken
	}

	return artistID, nil
}
package services

import (
	"context"
	"errors"
	"strings"

	"music-api/internal/models"
	"music-api/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNameRequired     = errors.New("user name is required")
	ErrUserEmailRequired    = errors.New("user email is required")
	ErrUserPasswordRequired = errors.New("user password is required")
	ErrUserPasswordTooShort = errors.New("user password must be at least 8 characters")
)

type UserService struct {
	Repository *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{
		Repository: repo,
	}
}

func (s *UserService) Register(
	ctx context.Context,
	name string,
	email string,
	password string,
) (models.User, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))
	password = strings.TrimSpace(password)

	if name == "" {
		return models.User{}, ErrUserNameRequired
	}

	if email == "" {
		return models.User{}, ErrUserEmailRequired
	}

	if password == "" {
		return models.User{}, ErrUserPasswordRequired
	}

	if len(password) < 8 {
		return models.User{}, ErrUserPasswordTooShort
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return models.User{}, err
	}

	user := models.User{
		Name:         name,
		Email:        email,
		PasswordHash: string(passwordHash),
	}

	return s.Repository.Create(ctx, user)
}

func (s *UserService) GetByEmail(
	ctx context.Context,
	email string,
) (models.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	return s.Repository.GetByEmail(ctx, email)
}

func (s *UserService) GetByID(
	ctx context.Context,
	id int,
) (models.User, error) {
	return s.Repository.GetByID(ctx, id)
}

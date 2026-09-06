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
	ErrUserAlreadyExists    = errors.New("an account with this email already exists")
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

	createdUser, err := s.Repository.Create(
		ctx,
		user,
	)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrUserEmailExists,
		) {
			return models.User{}, ErrUserAlreadyExists
		}

		return models.User{}, err
	}

	return createdUser, nil
}

func (s *UserService) GetByEmail(
	ctx context.Context,
	email string,
) (models.User, error) {
	email = strings.ToLower(
		strings.TrimSpace(email),
	)

	return s.Repository.GetByEmail(
		ctx,
		email,
	)
}

func (s *UserService) GetByID(
	ctx context.Context,
	id int,
) (models.User, error) {
	return s.Repository.GetByID(
		ctx,
		id,
	)
}

func (s *UserService) MarkEmailVerified(
	ctx context.Context,
	userID int,
) error {
	return s.Repository.MarkEmailVerified(
		ctx,
		userID,
	)
}

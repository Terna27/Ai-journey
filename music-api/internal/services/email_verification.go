package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"music-api/internal/models"
	"music-api/internal/repository"

	"github.com/jackc/pgx/v5"
)

var (
	ErrVerificationTokenInvalid = errors.New(
		"verification token is invalid or expired",
	)

	ErrEmailAlreadyVerified = errors.New(
		"email address is already verified",
	)
)

type EmailVerificationService struct {
	Repository   *repository.EmailVerificationRepository
	UserService  *UserService
	EmailService *EmailService
	FrontendURL  string
	TokenTTL     time.Duration
}

func NewEmailVerificationService(
	repository *repository.EmailVerificationRepository,
	userService *UserService,
	emailService *EmailService,
	frontendURL string,
) *EmailVerificationService {
	return &EmailVerificationService{
		Repository:   repository,
		UserService:  userService,
		EmailService: emailService,

		FrontendURL: strings.TrimRight(
			strings.TrimSpace(frontendURL),
			"/",
		),

		TokenTTL: time.Hour,
	}
}

func (s *EmailVerificationService) SendVerification(
	ctx context.Context,
	user models.User,
) error {
	if user.EmailVerified {
		return ErrEmailAlreadyVerified
	}

	rawToken, err :=
		generateVerificationToken()

	if err != nil {
		return err
	}

	tokenHash :=
		hashVerificationToken(
			rawToken,
		)

	// Only the newest verification link should
	// remain valid.
	if err := s.Repository.InvalidateForUser(
		ctx,
		user.ID,
	); err != nil {
		return err
	}

	expiresAt :=
		time.Now().Add(
			s.TokenTTL,
		)

	_, err = s.Repository.Create(
		ctx,
		user.ID,
		tokenHash,
		expiresAt,
	)

	if err != nil {
		return err
	}

	verificationURL, err :=
		s.buildVerificationURL(
			rawToken,
		)

	if err != nil {
		return err
	}

	if err :=
		s.EmailService.SendVerificationEmail(
			ctx,
			user.Email,
			user.Name,
			verificationURL,
		); err != nil {

		return err
	}

	return nil
}

func (s *EmailVerificationService) Verify(
	ctx context.Context,
	rawToken string,
) error {
	rawToken =
		strings.TrimSpace(
			rawToken,
		)

	if rawToken == "" {
		return ErrVerificationTokenInvalid
	}

	tokenHash :=
		hashVerificationToken(
			rawToken,
		)

	err := s.Repository.Verify(
		ctx,
		tokenHash,
	)

	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return ErrVerificationTokenInvalid
		}

		return err
	}

	return nil
}

func (s *EmailVerificationService) buildVerificationURL(
	rawToken string,
) (string, error) {
	if s.FrontendURL == "" {
		return "",
			errors.New(
				"frontend URL is required",
			)
	}

	parsedURL, err :=
		url.Parse(
			s.FrontendURL +
				"/verify-email",
		)

	if err != nil {
		return "", err
	}

	query :=
		parsedURL.Query()

	query.Set(
		"token",
		rawToken,
	)

	parsedURL.RawQuery =
		query.Encode()

	return parsedURL.String(), nil
}

func generateVerificationToken() (
	string,
	error,
) {
	randomBytes :=
		make(
			[]byte,
			32,
		)

	if _, err := rand.Read(
		randomBytes,
	); err != nil {

		return "",
			fmt.Errorf(
				"generate verification token: %w",
				err,
			)
	}

	return base64.RawURLEncoding.
			EncodeToString(
				randomBytes,
			),
		nil
}

func hashVerificationToken(
	rawToken string,
) string {
	hash :=
		sha256.Sum256(
			[]byte(rawToken),
		)

	return hex.EncodeToString(
		hash[:],
	)
}

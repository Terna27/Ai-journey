package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// -----------------------------------------------------------------
// Live audio provider abstraction (Phase 4.2)
// -----------------------------------------------------------------

// LiveAccessToken is a short-lived participant credential for
// one realtime audio room.
//
// The API secret never appears here: clients receive only the
// signed token and the public connection URL.
type LiveAccessToken struct {
	Token      string    `json:"token"`
	ConnectURL string    `json:"connect_url"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// LiveAudioProvider mints short-lived participant tokens for
// the realtime live audio provider (LiveKit/WebRTC).
//
// Live media NEVER flows through the Go HTTP server. This
// abstraction exists so the rest of the backend depends only
// on token minting, not on LiveKit implementation details.
type LiveAudioProvider interface {
	Configured() bool

	HostToken(
		roomName string,
		identity string,
	) (LiveAccessToken, error)

	ListenerToken(
		roomName string,
		identity string,
	) (LiveAccessToken, error)
}

var ErrLiveProviderNotConfigured = errors.New(
	"live audio provider is not configured",
)

const (
	HostLiveTokenTTL = 6 * time.Hour

	ListenerLiveTokenTTL = 4 * time.Hour
)

// LiveKitService mints LiveKit-compatible participant tokens.
//
// The API secret remains server-side only and is never returned
// directly to the client.
type LiveKitService struct {
	url       string
	apiKey    string
	apiSecret string
}

func NewLiveKitService(
	url string,
	apiKey string,
	apiSecret string,
) *LiveKitService {
	return &LiveKitService{
		url:       url,
		apiKey:    apiKey,
		apiSecret: apiSecret,
	}
}

func (s *LiveKitService) Configured() bool {
	return s.url != "" &&
		s.apiKey != "" &&
		s.apiSecret != ""
}

func (s *LiveKitService) HostToken(
	roomName string,
	identity string,
) (LiveAccessToken, error) {
	return s.mintToken(
		roomName,
		identity,
		HostLiveTokenTTL,
		true,
	)
}

func (s *LiveKitService) ListenerToken(
	roomName string,
	identity string,
) (LiveAccessToken, error) {
	return s.mintToken(
		roomName,
		identity,
		ListenerLiveTokenTTL,
		false,
	)
}

func (s *LiveKitService) mintToken(
	roomName string,
	identity string,
	ttl time.Duration,
	canPublish bool,
) (LiveAccessToken, error) {
	if !s.Configured() {
		return LiveAccessToken{},
			ErrLiveProviderNotConfigured
	}

	if roomName == "" || identity == "" {
		return LiveAccessToken{},
			errors.New(
				"room name and identity are required",
			)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(ttl)

	claims := jwt.MapClaims{
		"iss": s.apiKey,
		"sub": identity,
		"iat": now.Unix(),
		"nbf": now.Unix(),
		"exp": expiresAt.Unix(),

		"video": map[string]any{
			"roomJoin":       true,
			"room":           roomName,
			"canPublish":     canPublish,
			"canSubscribe":   true,
			"canPublishData": canPublish,
		},
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	signed, err := token.SignedString(
		[]byte(s.apiSecret),
	)
	if err != nil {
		return LiveAccessToken{}, fmt.Errorf(
			"failed to sign live access token: %w",
			err,
		)
	}

	return LiveAccessToken{
		Token:      signed,
		ConnectURL: s.url,
		ExpiresAt:  expiresAt,
	}, nil
}

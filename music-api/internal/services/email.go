package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var (
	ErrEmailAPIKeyRequired = errors.New("email API key is required")
	ErrEmailFromRequired   = errors.New("email sender address is required")
)

type EmailService struct {
	APIKey     string
	From       string
	HTTPClient *http.Client
}

func NewEmailService(
	apiKey string,
	from string,
) (*EmailService, error) {
	apiKey = strings.TrimSpace(apiKey)
	from = strings.TrimSpace(from)

	if apiKey == "" {
		return nil, ErrEmailAPIKeyRequired
	}

	if from == "" {
		return nil, ErrEmailFromRequired
	}

	return &EmailService{
		APIKey: apiKey,
		From:   from,
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}, nil
}

type resendEmailRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

func (s *EmailService) SendVerificationEmail(
	ctx context.Context,
	to string,
	name string,
	verificationURL string,
) error {
	body := resendEmailRequest{
		From: s.From,
		To: []string{
			to,
		},
		Subject: "Verify your email address",
		HTML: fmt.Sprintf(`
<!doctype html>
<html>
<body>
	<h2>Verify your email</h2>

	<p>Hello %s,</p>

	<p>
		Please verify your email address to finish
		setting up your Music account.
	</p>

	<p>
		<a href="%s">
			Verify email address
		</a>
	</p>

	<p>
		This verification link expires in 1 hour.
	</p>

	<p>
		If you did not create this account,
		you can ignore this email.
	</p>
</body>
</html>
`,
			htmlEscape(name),
			htmlEscape(verificationURL),
		),
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.resend.com/emails",
		bytes.NewReader(payload),
	)
	if err != nil {
		return err
	}

	req.Header.Set(
		"Authorization",
		"Bearer "+s.APIKey,
	)

	req.Header.Set(
		"Content-Type",
		"application/json",
	)

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 ||
		resp.StatusCode >= 300 {

		responseBody, readErr :=
			io.ReadAll(
				io.LimitReader(
					resp.Body,
					4096,
				),
			)

		if readErr != nil {
			return fmt.Errorf(
				"resend returned status %d",
				resp.StatusCode,
			)
		}

		return fmt.Errorf(
			"resend returned status %d: %s",
			resp.StatusCode,
			strings.TrimSpace(
				string(responseBody),
			),
		)
	}

	return nil
}

func htmlEscape(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&#34;",
		"'", "&#39;",
	)

	return replacer.Replace(value)
}

package brevo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"somotracker/internal/platform/apperr"
)

// CodeSender is the interface for sending OTP codes via email.
type CodeSender interface {
	SendCode(ctx context.Context, email string, code string) error
}

// Sender implements CodeSender using Brevo's REST API.
type Sender struct {
	apiKey      string
	senderEmail string
	senderName  string
	client      *http.Client
}

// New creates a Brevo email sender.
func New(apiKey, senderEmail, senderName string) *Sender {
	return &Sender{
		apiKey:      apiKey,
		senderEmail: senderEmail,
		senderName:  senderName,
		client:      &http.Client{Timeout: 10 * time.Second},
	}
}

// SendCode sends an OTP code via Brevo transactional email.
func (s *Sender) SendCode(ctx context.Context, email string, code string) error {
	payload := map[string]any{
		"sender": map[string]string{
			"name":  s.senderName,
			"email": s.senderEmail,
		},
		"to": []map[string]string{
			{
				"email": email,
				"name":  email,
			},
		},
		"subject": "Your Somotracker verification code",
		"htmlContent": fmt.Sprintf(
			`<p>Your verification code is <strong>%s</strong>. It expires in 10 minutes.</p><p>Do not share this code with anyone.</p>`,
			code,
		),
		"textContent": fmt.Sprintf(
			"Your verification code is %s. It expires in 10 minutes. Do not share this code with anyone.",
			code,
		),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return apperr.Wrap(apperr.INTERNAL_ERROR, "marshal email payload", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.brevo.com/v3/smtp/email", bytes.NewReader(body))
	if err != nil {
		return apperr.Wrap(apperr.INTERNAL_ERROR, "create request", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return apperr.Wrap(apperr.INTERNAL_ERROR, "send email request", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return apperr.New(apperr.INTERNAL_ERROR, fmt.Sprintf("brevo API returned status %d: %s", resp.StatusCode, string(respBody)))
	}

	return nil
}

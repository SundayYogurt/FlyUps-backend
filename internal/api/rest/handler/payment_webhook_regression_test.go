package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"flyup/internal/services"

	"github.com/gofiber/fiber/v3"
)

type webhookServiceStub struct {
	services.InvestmentService
	err error
}

func (s webhookServiceStub) HandleStripeWebhook([]byte, string) error { return s.err }

func TestWebhookHTTPStatusDistinguishesBadPayloadAndDatabaseFailure(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
	}{
		{"database failure", errors.New("database unavailable"), http.StatusInternalServerError},
		{"invalid signature", services.ErrInvalidStripeWebhook, http.StatusBadRequest},
		{"processed", nil, http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			h := &InvestmentHandler{svc: webhookServiceStub{err: tc.err}}
			app.Post("/stripe/webhook", h.StripeWebhook)
			response, err := app.Test(httptest.NewRequest(http.MethodPost, "/stripe/webhook", strings.NewReader(`{}`)))
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if response.StatusCode != tc.status {
				t.Fatalf("got HTTP %d, want %d", response.StatusCode, tc.status)
			}
		})
	}
}

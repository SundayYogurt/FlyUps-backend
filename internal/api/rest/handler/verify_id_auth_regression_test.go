package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"flyup/config"
	"flyup/internal/api/rest"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/helper"
	"flyup/internal/repository"
	"flyup/internal/services"

	"github.com/gofiber/fiber/v3"
)

type verifyIDAuthRepo struct{ repository.UserRepository }

func (verifyIDAuthRepo) FindUserById(id uint) (*domain.User, error) {
	return &domain.User{ID: id, Email: "test@example.com", Role: "pioneer"}, nil
}

type verifyIDAuthService struct {
	services.UserService
	verifiedUserID uint
}

func (s *verifyIDAuthService) VerifyID(userID uint, _ dto.VerifyIDInput) error {
	s.verifiedUserID = userID
	return nil
}

func (s *verifyIDAuthService) GetKYCSessionByToken(_ context.Context, token string) (*domain.KYCUploadSession, error) {
	if token == "kyc-session" {
		return &domain.KYCUploadSession{UserID: 9, Status: domain.VerifyStatusPending, ExpiresAt: time.Now().Add(time.Minute)}, nil
	}
	return nil, errors.New("unknown kyc token")
}

func (s *verifyIDAuthService) MarkKYCSessionCompleted(context.Context, string) error { return nil }

func TestVerifyIDAcceptsBearerAndKYCSession(t *testing.T) {
	auth := helper.SetupAuth("test-secret-key-for-verify-id-auth")
	middleware := rest.SetupMiddleware(auth, verifyIDAuthRepo{})
	for _, tc := range []struct {
		name       string
		path       string
		bearer     bool
		invalidJWT bool
		wantID     uint
		wantStatus int
	}{
		{name: "bearer", path: "/user/id-verify", bearer: true, wantID: 7, wantStatus: http.StatusOK},
		{name: "kyc session", path: "/user/id-verify?token=kyc-session", wantID: 9, wantStatus: http.StatusOK},
		{name: "missing auth", path: "/user/id-verify", wantStatus: http.StatusUnauthorized},
		{name: "invalid bearer", path: "/user/id-verify", invalidJWT: true, wantStatus: http.StatusUnauthorized},
		{name: "invalid kyc session", path: "/user/id-verify?token=unknown", wantStatus: http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := &verifyIDAuthService{}
			handler := NewUserHandler(svc, auth, nil, config.AppConfig{}, nil)
			app := fiber.New()
			app.Post("/user/id-verify", requireUserUnlessKYCSession(middleware.Authorize), handler.VerifyIDCard)
			req := httptest.NewRequest(http.MethodPost, tc.path, bytes.NewBufferString(`{"id_card_url":"https://example.com/card.jpg","selfie_url":"https://example.com/selfie.jpg","declare_truth":true}`))
			req.Header.Set("Content-Type", "application/json")
			if tc.bearer {
				jwt, err := auth.GenerateToken(tc.wantID, "test@example.com", "pioneer")
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Set("Authorization", "Bearer "+jwt)
			}
			if tc.invalidJWT {
				req.Header.Set("Authorization", "Bearer invalid")
			}
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tc.wantStatus || svc.verifiedUserID != tc.wantID {
				t.Fatalf("status=%d verifiedUserID=%d, want %d and %d", resp.StatusCode, svc.verifiedUserID, tc.wantStatus, tc.wantID)
			}
		})
	}
}

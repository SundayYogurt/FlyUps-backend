package rest

import (
	"encoding/json"
	"flyup/internal/dto"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestComplaintInputBoundaries(t *testing.T) {
	app := fiber.New()
	app.Post("/", func(c fiber.Ctx) error {
		var req dto.CreateComplaintRequest
		if err := BindJSON(c, &req); err != nil {
			return c.SendStatus(400)
		}
		if err := validator.New().Struct(req); err != nil {
			return c.SendStatus(400)
		}
		return c.SendStatus(204)
	})
	for _, tc := range []struct {
		name, subject, body, evidence string
		status                        int
	}{
		{"min", "กกก", strings.Repeat("ก", 10), "https://example.com/proof", 204},
		{"below min", "กก", strings.Repeat("ก", 10), "https://example.com/proof", 400},
		{"max", strings.Repeat("ก", 200), strings.Repeat("ก", 5000), "https://example.com/proof", 204},
		{"subject too long", strings.Repeat("ก", 201), strings.Repeat("ก", 10), "https://example.com/proof", 400},
		{"body too long", "กกก", strings.Repeat("ก", 5001), "https://example.com/proof", 400},
		{"blank subject", "   ", strings.Repeat("ก", 10), "https://example.com/proof", 400},
		{"blank body", "กกก", strings.Repeat(" ", 10), "https://example.com/proof", 400},
		{"padded short subject", " กก ", strings.Repeat("ก", 10), "https://example.com/proof", 400},
		{"padded short body", "กกก", " 123456789 ", "https://example.com/proof", 400},
		{"ftp", "กกก", strings.Repeat("ก", 10), "ftp://example.com/proof", 400},
		{"credentials", "กกก", strings.Repeat("ก", 10), "https://user:pass@example.com/proof", 400},
		{"URL max", "กกก", strings.Repeat("ก", 10), "https://example.com/" + strings.Repeat("a", 2048-len("https://example.com/")), 204},
		{"URL over max", "กกก", strings.Repeat("ก", 10), "https://example.com/" + strings.Repeat("a", 2049-len("https://example.com/")), 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(dto.CreateComplaintRequest{ProjectID: 1, Subject: tc.subject, Body: tc.body, Evidence: tc.evidence})
			req := httptest.NewRequest("POST", "/", strings.NewReader(string(body)))
			req.Header.Set("Content-Type", "application/json")
			res, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			res.Body.Close()
			if res.StatusCode != tc.status {
				t.Fatalf("got %d want %d", res.StatusCode, tc.status)
			}
		})
	}
}

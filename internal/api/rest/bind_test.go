package rest

import (
	"bytes"
	"flyup/internal/dto"
	"github.com/gofiber/fiber/v3"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBindRejectsInvalidInputs(t *testing.T) {
	app := fiber.New()
	app.Post("/", func(c fiber.Ctx) error {
		var input dto.UpdateProjectRequest
		if err := BindBody(c, &input); err != nil {
			return c.SendStatus(400)
		}
		return c.SendStatus(204)
	})
	app.Get("/", func(c fiber.Ctx) error {
		var input dto.PublicProjectFilter
		if err := BindQuery(c, &input); err != nil {
			return c.SendStatus(400)
		}
		return c.SendStatus(204)
	})
	for _, tc := range []struct {
		method, path, body, contentType string
		status                          int
	}{
		{"GET", "/?min_goal=NaN", "", "", 400},
		{"GET", "/?max_goal=Inf", "", "", 400},
		{"GET", "/?min_goal=100", "", "", 204},
		{"POST", "/", `{"funding_goal":1e999}`, "application/json", 400},
		{"POST", "/", `{"title":"` + strings.Repeat("ก", 51) + `"}`, "application/json", 400},
		{"POST", "/", "funding_goal=NaN", "application/x-www-form-urlencoded", 400},
		{"POST", "/", `{"funding_goal":1000}`, "application/json", 204},
	} {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		req.Header.Set("Content-Type", tc.contentType)
		res, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != tc.status {
			t.Errorf("%s %s %s: got %d want %d", tc.method, tc.path, tc.body, res.StatusCode, tc.status)
		}
	}
}

func TestBindFormUsesJSONNames(t *testing.T) {
	for _, contentType := range []string{"application/x-www-form-urlencoded", "multipart/form-data"} {
		for _, value := range []string{"1000", "NaN", "+Inf", "1000000001"} {
			t.Run(contentType+value, func(t *testing.T) {
				app := fiber.New()
				app.Post("/", func(c fiber.Ctx) error {
					var input dto.UpdateProjectRequest
					if err := BindBody(c, &input); err != nil {
						return c.SendStatus(400)
					}
					if input.FundingGoal == nil || *input.FundingGoal != 1000 {
						t.Errorf("form field was lost: %+v", input)
					}
					return c.SendStatus(204)
				})
				body := new(bytes.Buffer)
				ct := contentType
				if contentType == "multipart/form-data" {
					writer := multipart.NewWriter(body)
					_ = writer.WriteField("funding_goal", value)
					_ = writer.Close()
					ct = writer.FormDataContentType()
				} else {
					body.WriteString("funding_goal=" + value)
				}
				req := httptest.NewRequest("POST", "/", body)
				req.Header.Set("Content-Type", ct)
				res, err := app.Test(req)
				if err != nil {
					t.Fatal(err)
				}
				defer res.Body.Close()
				want := 400
				if value == "1000" {
					want = 204
				}
				if res.StatusCode != want {
					t.Fatalf("got %d, want %d", res.StatusCode, want)
				}
			})
		}
	}
}

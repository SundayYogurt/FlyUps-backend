package main

import (
	"bytes"
	"flyup/internal/api"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheck(t *testing.T) {

	tests := []struct {
		Name           string
		Path           string
		Method         string
		ExpectedStatus int
		ExpectedBody   string
	}{
		{
			Name:           "health Check",
			Path:           "/",
			Method:         "GET",
			ExpectedStatus: 200,
			ExpectedBody:   `{"message":"Healthy"}`,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			app := fiber.New()
			app.Get(test.Path, api.HealthCheck)
			req := httptest.NewRequest(test.Method, test.Path, nil)
			res, _ := app.Test(req)
			assert.Equal(t, test.ExpectedStatus, res.StatusCode, test.Name)
			buffer := new(bytes.Buffer)
			_, err := buffer.ReadFrom(res.Body)
			assert.NoError(t, err)
			assert.Equal(t, test.ExpectedBody, buffer.String())
		})
	}
}

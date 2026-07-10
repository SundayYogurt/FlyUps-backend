//go:build integration

package integration

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

const testAppSecret = "integration-test-secret-please-change-32chars"

func testDB(t *testing.T) *gorm.DB {
	t.Helper()

	_ = godotenv.Load("../../.env")

	dsn := os.Getenv("TEST_DSN")
	if dsn == "" {
		dsn = os.Getenv("DSN")
	}
	if dsn == "" {
		t.Fatal("Test_DSN (or DSN) env var is required to run integration tests")
	}
}

//go:build integration

package integration

import "testing"

func TestCreateProject_Success(t *testing.T) {
	tx := beginTx(t)
	app, auth := newTestApp(tx)

	pioneer := createVerifiedPioneer(t, tx)
	token, err := auth.GenerateToken(pioneer.ID, pioneer.Email, pioneer.Role)
}

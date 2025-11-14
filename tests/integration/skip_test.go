//go:build !integration

package integration_test

import "testing"

func TestIntegrationSkipped(t *testing.T) {
	t.Skip("integration tests require -tags=integration")
}

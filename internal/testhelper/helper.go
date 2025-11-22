package testhelper

import (
	stdhttp "net/http"
	"os"
	"strings"
	"testing"
)

const (
	TestOnlineAPIURL = "https://api.restful-api.dev/objects"
)

func PerformIntegrationTests(t *testing.T) {
	t.Helper()
	if val := os.Getenv("PERFORM_INTEGRATION_TEST"); !strings.EqualFold(val, "true") {
		t.Skip("skipping integration test")
	}
}

type MockRoundTripper struct {
	Fn func(req *stdhttp.Request) (*stdhttp.Response, error)
}

func (m MockRoundTripper) RoundTrip(req *stdhttp.Request) (*stdhttp.Response, error) {
	return m.Fn(req)
}

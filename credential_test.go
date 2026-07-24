package lakefs

import (
	"os"
	"strings"
	"testing"
	"time"
)

func integrationClient(t *testing.T) *Client {
	t.Helper()

	if os.Getenv("LAKEFS_INTEGRATION") != "1" {
		t.Skip("set LAKEFS_INTEGRATION=1 to run lakeFS integration tests")
	}
	endpoint := os.Getenv("LAKEFS_ENDPOINT")
	accessKeyID := os.Getenv("LAKEFS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("LAKEFS_SECRET_ACCESS_KEY")
	if endpoint == "" || accessKeyID == "" || secretAccessKey == "" {
		t.Skip("lakeFS integration environment is not configured")
	}
	if !strings.Contains(endpoint, "://") {
		endpoint = "http://" + endpoint
	}

	return NewClient(&BasicAuth{
		Endpoint:        endpoint,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
	}, &Options{Timeout: 30 * time.Second})
}

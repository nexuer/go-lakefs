package lakefs

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/nexuer/ghttp"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestRewriteAllObjectMetadataRequest(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/api/v1/repositories/repo/branches/main/objects/stat/user_metadata" {
			t.Errorf("path = %s", r.URL.Path)
		}
		got := r.URL.Query()
		want := url.Values{"path": []string{"dir/file.txt"}}
		if got.Encode() != want.Encode() {
			t.Errorf("query = %q, want %q", got.Encode(), want.Encode())
		}
		username, password, ok := r.BasicAuth()
		if !ok || username != "access" || password != "secret" {
			t.Error("basic auth was not applied")
		}
		var body RewriteAllObjectMetadataOptions
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.Path != "dir/file.txt" || body.Set["owner"] != "sdk" {
			t.Errorf("body = %+v", body)
		}
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Header:     make(http.Header),
			Body:       io.NopCloser(http.NoBody),
			Request:    r,
		}, nil
	})

	client := NewClient(&BasicAuth{
		Endpoint:        "http://lakefs.test",
		AccessKeyID:     "access",
		SecretAccessKey: "secret",
	})
	client.cc = ghttp.NewClient(
		ghttp.WithEndpoint("http://lakefs.test"),
		ghttp.WithTransport(transport),
	)
	err := client.Objects.RewriteAllObjectMetadata(context.Background(), "repo", "main", &RewriteAllObjectMetadataOptions{
		Path: "dir/file.txt",
		Set:  ObjectUserMetadata{"owner": "sdk"},
	})
	if err != nil {
		t.Fatal(err)
	}
}

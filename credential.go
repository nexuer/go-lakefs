package lakefs

import (
	"net/http"
)

type Credential interface {
	GetEndpoint() string
	BeforeRequest(req *http.Request)
}

type BasicAuth struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
}

func (b BasicAuth) GetEndpoint() string {
	return b.Endpoint
}

func (b BasicAuth) BeforeRequest(req *http.Request) {
	req.SetBasicAuth(b.AccessKeyID, b.SecretAccessKey)
}

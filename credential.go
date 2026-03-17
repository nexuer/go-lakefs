package lakefs

import (
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type Credential interface {
	GetEndpoint() string
	BeforeRequest(req *http.Request)
	CredentialsProvider() aws.CredentialsProvider
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

func (b BasicAuth) CredentialsProvider() aws.CredentialsProvider {
	return credentials.NewStaticCredentialsProvider(b.AccessKeyID, b.SecretAccessKey, "")
}

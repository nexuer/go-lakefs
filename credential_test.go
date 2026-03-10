package lakefs

import "os"

var testBasicAuth = &BasicAuth{
	Endpoint:        "127.0.0.1:38000",
	AccessKeyID:     os.Getenv("LAKEFS_ACCESS_KEY_ID"),
	SecretAccessKey: os.Getenv("LAKEFS_SECRET_ACCESS_KEY"),
}

func testCredential() Credential {
	return testBasicAuth
}

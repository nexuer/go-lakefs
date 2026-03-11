package lakefs

import "os"

var testBasicAuth = &BasicAuth{
	Endpoint:        os.Getenv("LAKEFS_ENDPOINT"),
	AccessKeyID:     os.Getenv("LAKEFS_ACCESS_KEY_ID"),
	SecretAccessKey: os.Getenv("LAKEFS_SECRET_ACCESS_KEY"),
}

func testCredential() Credential {
	return testBasicAuth
}

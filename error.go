package lakefs

import (
	"errors"
	"net/http"

	"github.com/nexuer/ghttp"
)

type responseError interface {
	HTTPStatusCode() int
}

func s3StatusCode(err error) int {
	var apiErr responseError
	if errors.As(err, &apiErr) {
		return apiErr.HTTPStatusCode()
	}
	return 0
}

func StatusCode(err error) int {
	if err == nil {
		return http.StatusOK
	}
	code, ok := ghttp.StatusCode(err)
	if ok {
		return code
	}
	return s3StatusCode(err)
}

func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return StatusCode(err) == http.StatusNotFound
}

func newStatusCode(code int, err error) error {
	if err == nil {
		return nil
	}
	return &ghttp.Error{
		StatusCode: http.StatusNotFound,
		Err:        err,
	}
}

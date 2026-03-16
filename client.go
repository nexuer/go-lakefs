package lakefs

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/nexuer/ghttp"
)

type service struct {
	client *Client
}

type Options struct {
	UserAgent string
	Timeout   time.Duration
	Proxy     func(*http.Request) (*url.URL, error)
	Debug     bool
	TLS       *tls.Config
	Limiter   ghttp.Limiter
}

type Client struct {
	cc *ghttp.Client

	apiVersion APIVersion
	credential Credential

	common service

	Repositories *Repositories
	Branches     *Branches
	Commits      *Commits
	Refs         *Refs
	Objects      *Objects
	Tags         *Tags
	Staging      *Staging
}

func NewClient(credential Credential, opts ...*Options) *Client {
	c := &Client{
		apiVersion: V1,
	}

	clientOpts := c.parseOptions(opts...)
	clientOpts = append(clientOpts, ghttp.WithNot2xxError(func() error {
		return new(Error)
	}))
	c.cc = ghttp.NewClient(clientOpts...)
	c.common.client = c

	c.Repositories = (*Repositories)(&c.common)
	c.Branches = (*Branches)(&c.common)
	c.Refs = (*Refs)(&c.common)
	c.Commits = (*Commits)(&c.common)
	c.Objects = (*Objects)(&c.common)
	c.Tags = (*Tags)(&c.common)
	c.Staging = (*Staging)(&c.common)

	c.SetCredential(credential)
	return c
}

func (c *Client) SetCredential(credential Credential) {
	var endpoint string
	if credential != nil {
		endpoint = credential.GetEndpoint()
	}

	c.cc.SetEndpoint(endpoint)
	c.credential = credential

}

func (c *Client) parseOptions(opts ...*Options) []ghttp.ClientOption {
	var opt *Options
	if len(opts) > 0 && opts[0] != nil {
		opt = opts[0]
	} else {
		opt = new(Options)
	}

	clientOpts := make([]ghttp.ClientOption, 0)

	if opt.UserAgent != "" {
		clientOpts = append(clientOpts, ghttp.WithUserAgent(opt.UserAgent))
	}

	if opt.Debug {
		clientOpts = append(clientOpts, ghttp.WithDebug(opt.Debug))
	}

	if opt.Timeout > 0 {
		clientOpts = append(clientOpts, ghttp.WithTimeout(opt.Timeout))
	}

	if opt.Proxy != nil {
		clientOpts = append(clientOpts, ghttp.WithProxy(opt.Proxy))
	}

	if opt.TLS != nil {
		clientOpts = append(clientOpts, ghttp.WithTLSConfig(opt.TLS))
	}

	if opt.Limiter != nil {
		clientOpts = append(clientOpts, ghttp.WithLimiter(opt.Limiter))
	}

	return clientOpts
}

func (c *Client) API(path string, ver ...APIVersion) string {
	if len(ver) > 0 && ver[0] != "" {
		return fmt.Sprintf("/api/%s/%s", ver[0], path)
	}
	return fmt.Sprintf("/api/%s/%s", c.apiVersion, path)
}

func (c *Client) InvokeWithCredential(ctx context.Context, method, path string, args any, reply any, fn ...ghttp.RequestFunc) (*http.Response, error) {
	if c.credential == nil {
		return nil, errors.New("credential is nil")
	}
	fns := make([]ghttp.RequestFunc, 1, len(fn)+1)
	fns[0] = func(request *http.Request) error {
		c.credential.BeforeRequest(request)
		return nil
	}
	fns = append(fns, fn...)
	return c.Invoke(ctx, method, c.API(path), args, reply, fns...)
}

func (c *Client) Invoke(ctx context.Context, method, path string, args any, reply any, fn ...ghttp.RequestFunc) (*http.Response, error) {
	opts := &ghttp.CallOptions{
		BeforeHooks: fn,
	}
	if (method == http.MethodGet || method == http.MethodHead) && args != nil {
		opts.Query = args
		args = nil
	}

	return c.cc.Invoke(ctx, method, path, args, reply, opts)
}

func (c *Client) DoWithCredential(req *http.Request, fn ...ghttp.RequestFunc) (*http.Response, error) {
	if c.credential == nil {
		return nil, errors.New("credential is nil")
	}
	fns := make([]ghttp.RequestFunc, 1, len(fn)+1)
	fns[0] = func(request *http.Request) error {
		c.credential.BeforeRequest(request)
		return nil
	}
	fns = append(fns, fn...)
	return c.Do(req, fns...)
}

func (c *Client) Do(req *http.Request, fn ...ghttp.RequestFunc) (*http.Response, error) {
	opts := &ghttp.CallOptions{
		BeforeHooks: fn,
	}

	return c.cc.Do(req, opts)
}

package lakefs

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/nexuer/ghttp"
)

type Staging service

type StagingLocation struct {
	PhysicalAddress    string     `json:"physical_address"`
	PresignedUrl       string     `json:"presigned_url"`
	PresignedUrlExpiry *Timestamp `json:"presigned_url_expiry"`
}

type GetBackingOptions struct {
	Presign bool   `url:"presign,omitempty"`
	Path    string `url:"path,omitempty"`
}

func (s *Staging) GetBacking(ctx context.Context, repository, branch string, opts *GetBackingOptions) (*StagingLocation, error) {
	u := fmt.Sprintf("repositories/%s/branches/%s/staging/backing", repository, branch)
	var row StagingLocation
	_, err := s.client.InvokeWithCredential(ctx, http.MethodGet, u, opts, &row)
	return &row, err
}

type PutBackingOptions struct {
	IfNoneMatch     string           `json:"-"`
	IfMatch         string           `json:"-"`
	Path            string           `json:"-"`
	StagingMetadata *StagingMetadata `json:"-"`
}

func (p *PutBackingOptions) Request() []ghttp.RequestFunc {
	if p == nil {
		return nil
	}
	return []ghttp.RequestFunc{
		func(request *http.Request) error {
			if p.IfNoneMatch != "" {
				request.Header.Set("If-None-Match", p.IfNoneMatch)
			}
			if p.IfMatch != "" {
				request.Header.Set("If-Match", p.IfMatch)
			}
			return nil
		},
	}
}

type StagingMetadata struct {
	Staging      *StagingLocation  `json:"staging"`
	Checksum     string            `json:"checksum"`
	SizeBytes    int64             `json:"size_bytes"`
	UserMetadata map[string]string `json:"user_metadata"`
	ContentType  string            `json:"content_type"`
	Mtime        int64             `json:"mtime"`
	Force        bool              `json:"force"`
}

func (s *Staging) PutBacking(ctx context.Context, repository, branch string, opts *PutBackingOptions) (*ObjectStats, error) {
	if opts == nil {
		opts = &PutBackingOptions{}
	}
	u := fmt.Sprintf("repositories/%s/branches/%s/staging/backing", repository, branch)
	if opts != nil && opts.Path != "" {
		u += "?path=" + url.QueryEscape(opts.Path)
	}
	var row ObjectStats
	_, err := s.client.InvokeWithCredential(ctx, http.MethodPut, u, opts.StagingMetadata, &row, opts.Request()...)
	return &row, err
}

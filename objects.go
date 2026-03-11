package lakefs

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nexuer/ghttp"
)

type Objects service

type ObjectUserMetadata map[string]string

type RangeByteSize struct {
	Start, End int64
}

func (r RangeByteSize) String() string {
	return fmt.Sprintf("bytes=%d-%d", r.Start, r.End)
}

type GetObjectContentOptions struct {
	Range       *RangeByteSize `url:"-"`
	IfNoneMatch string         `url:"-"`

	Path    string `url:"path,omitempty"`
	Presign *bool  `url:"presign,omitempty"`
}

func (g *GetObjectContentOptions) Request() []ghttp.RequestFunc {
	if g == nil {
		return nil
	}
	if g.Range == nil && g.IfNoneMatch == "" {
		return nil
	}
	return []ghttp.RequestFunc{
		func(request *http.Request) error {
			if g.IfNoneMatch != "" {
				request.Header.Set("If-None-Match", g.IfNoneMatch)
			}
			if g.Range != nil {
				request.Header.Set("Range", g.Range.String())
			}

			return nil
		},
	}
}

type ObjectHeaders struct {
	ContentLength int64
	LastModified  time.Time
	ETag          string
	ContentRange  string
}

func NewObjectHeaders(resp *http.Response) ObjectHeaders {
	var lastModified time.Time
	lastModifiedStr := resp.Header.Get("Last-Modified")
	if lastModifiedStr != "" {
		lastModified, _ = time.Parse(time.RFC1123, lastModifiedStr)
	}
	return ObjectHeaders{
		ContentLength: resp.ContentLength,
		LastModified:  lastModified,
		ContentRange:  resp.Header.Get("Content-Range"),
		ETag:          strings.Trim(resp.Header.Get("ETag"), `"`),
	}
}

type ObjectContent struct {
	Body io.ReadCloser

	Headers ObjectHeaders
}

func (o *Objects) GetObjectContent(ctx context.Context, repository, ref string, opts *GetObjectContentOptions) (*ObjectContent, error) {
	u := fmt.Sprintf("repositories/%s/refs/%s/objects", repository, ref)
	resp, err := o.client.InvokeWithCredential(ctx, http.MethodGet, u, opts, nil, opts.Request()...)
	if err != nil {
		return nil, err
	}
	headers := NewObjectHeaders(resp)
	oc := &ObjectContent{
		Body:    resp.Body,
		Headers: headers,
	}

	return oc, nil
}

type ObjectExistsOptions struct {
	Range *RangeByteSize `url:"-"`

	Path string `url:"path,omitempty"`
}

func (g *ObjectExistsOptions) Request() []ghttp.RequestFunc {
	if g == nil {
		return nil
	}
	if g.Range == nil {
		return nil
	}
	return []ghttp.RequestFunc{
		func(request *http.Request) error {
			if g.Range != nil {
				request.Header.Set("Range", g.Range.String())
			}

			return nil
		},
	}
}

func (o *Objects) ObjectExists(ctx context.Context, repository, ref string, opts *ObjectExistsOptions) (bool, *ObjectHeaders, error) {
	u := fmt.Sprintf("repositories/%s/refs/%s/objects", repository, ref)
	resp, err := o.client.InvokeWithCredential(ctx, http.MethodHead, u, opts, nil, opts.Request()...)
	if err != nil {
		code, _ := ghttp.StatusForErr(err)
		if code == http.StatusNotFound {
			return false, nil, nil
		}
		return false, nil, err
	}
	headers := NewObjectHeaders(resp)
	return true, &headers, nil
}

type CreateObjectOptions struct {
	IfNoneMatch string `url:"-"`
	IfMatch     string `url:"-"`

	Force bool   `url:"force,omitempty"`
	Path  string `url:"path,omitempty"`
}

func (c *CreateObjectOptions) Request() []ghttp.RequestFunc {
	if c == nil {
		return nil
	}
	return []ghttp.RequestFunc{
		func(request *http.Request) error {
			if c.IfMatch != "" {
				request.Header.Set("If-Match", c.IfMatch)
			}

			if c.IfNoneMatch != "" {
				request.Header.Set("If-None-Match", c.IfNoneMatch)
			}

			return nil
		},
	}
}

type ObjectStats struct {
	Path                  string             `json:"path"`
	PathType              PathType           `json:"path_type"`
	PhysicalAddress       string             `json:"physical_address"`
	PhysicalAddressExpiry int64              `json:"physical_address_expiry"`
	Checksum              string             `json:"checksum"`
	SizeBytes             int64              `json:"size_bytes"`
	Mtime                 int64              `json:"mtime"`
	Metadata              ObjectUserMetadata `json:"metadata"`
	ContentType           string             `json:"content_type"`
}

type ListObjectOptions struct {
	ListOptions `url:",inline"`

	UserMetadata *bool  `url:"user_metadata,omitempty"`
	Presign      *bool  `url:"presign,omitempty"`
	Delimiter    string `url:"delimiter,omitempty"`
	Prefix       string `url:"prefix,omitempty"`
}

func (o *Objects) ListObjects(ctx context.Context, repository, ref string, opts *ListObjectOptions) (*Records[ObjectStats], error) {
	u := fmt.Sprintf("repositories/%s/refs/%s/objects/ls", repository, ref)
	var records Records[ObjectStats]
	_, err := o.client.InvokeWithCredential(ctx, http.MethodGet, u, opts, &records)
	return &records, err
}

// CreateObject
// todo: 待完善
// 适用场景：单次请求上传一个完整文件
// 大文件或要求性能推荐走 S3 Gateway，用 aws-sdk-go-v2 的 manager.Uploader，它会自动处理分块、并发上传、失败重试等
func (o *Objects) CreateObject(ctx context.Context, repository, branch string, opts *CreateObjectOptions) (*ObjectStats, error) {
	u := fmt.Sprintf("repositories/%s/branches/%s/objects", repository, branch)
	//w := multipart.NewWriter()
	var reply ObjectStats
	_, err := o.client.InvokeWithCredential(ctx, http.MethodPost, u, opts, &reply, opts.Request()...)
	if err != nil {
		return nil, err
	}
	return &reply, nil
}

type DeleteObjectOptions struct {
	Force       bool   `url:"force,omitempty"`
	NoTombstone bool   `url:"no_tombstone,omitempty"`
	Path        string `url:"path,omitempty"`
}

func (o *Objects) DeleteObject(ctx context.Context, repository, branch string, opts *DeleteObjectOptions) error {
	u := fmt.Sprintf("repositories/%s/branches/%s/objects", repository, branch)

	_, err := o.client.InvokeWithCredential(ctx, http.MethodDelete, u, nil, nil, func(request *http.Request) error {
		return ghttp.SetQuery(request, opts)
	})
	return err
}

package lakefs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

	Path string `url:"path,omitempty"`

	Transfer *TransferOptions `url:"-"`
}

func (g *GetObjectContentOptions) setS3Options(input *transfermanager.GetObjectInput) {
	if g == nil {
		return
	}
	if g.IfNoneMatch != "" {
		input.IfNoneMatch = aws.String(g.IfNoneMatch)
	}
}

func (g *GetObjectContentOptions) setS3Options0(input *s3.GetObjectInput) {
	if g == nil {
		return
	}
	if g.IfNoneMatch != "" {
		input.IfNoneMatch = aws.String(g.IfNoneMatch)
	}
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
	s3cc := o.client.s3()
	if s3cc != nil {
		return s3cc.GetObjectContent(ctx, repository, ref, opts)
	}

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

type ObjectStats struct {
	Path                  string             `json:"path"`
	PathType              PathType           `json:"path_type"`
	PhysicalAddress       string             `json:"physical_address"`
	PhysicalAddressExpiry int64              `json:"physical_address_expiry"`
	Checksum              string             `json:"checksum"`
	SizeBytes             int64              `json:"size_bytes"`
	Mtime                 *Timestamp         `json:"mtime"`
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
	if err != nil {
		return nil, err
	}
	return &records, err
}

type CreateObjectOptions struct {
	IfNoneMatch string `url:"-"`
	IfMatch     string `url:"-"`

	Force bool   `url:"force,omitempty"`
	Path  string `url:"path,omitempty"`

	Transfer *TransferOptions `url:"-"`
}

func (c *CreateObjectOptions) setS3Options(input *transfermanager.UploadObjectInput) {
	if c == nil {
		return
	}
	if c.IfNoneMatch != "" {
		input.IfNoneMatch = aws.String(c.IfNoneMatch)
	}
	if c.IfMatch != "" {
		input.IfMatch = aws.String(c.IfMatch)
	}
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

// CreateObject 直接上传文件到lakeFS
// 如果你追求极致的上传性能（尤其是针对 TB 级数据或高并发场景），请走预签名 (Pre-signed) 模式
func (o *Objects) CreateObject(ctx context.Context, repository, branch string, reader io.Reader, opts *CreateObjectOptions) (*ObjectStats, error) {
	s3cc := o.client.s3()
	if s3cc != nil {
		err := s3cc.CreateObject(ctx, repository, branch, reader, opts)
		if err != nil {
			return nil, err
		}
		return o.GetObjectMetadata(ctx, repository, branch, &GetObjectMetadataOptions{
			Path: opts.Path,
		})
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("content", opts.Path)
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(part, reader)
	if err != nil {
		return nil, err
	}

	if err = writer.Close(); err != nil {
		return nil, err
	}

	u := o.client.API(fmt.Sprintf("repositories/%s/branches/%s/objects", repository, branch))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, &body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	err = ghttp.SetQuery(req, opts)
	if err != nil {
		return nil, err
	}
	resp, err := o.client.DoWithCredential(req, opts.Request()...)
	if err != nil {
		return nil, err
	}
	var reply ObjectStats
	if err = ghttp.BindResponseBody(resp, &reply); err != nil {
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

type GetObjectMetadataOptions struct {
	Path         string `url:"path,omitempty"`
	UserMetadata *bool  `url:"user_metadata,omitempty"`
	Presign      *bool  `url:"presign,omitempty"`
}

func (o *Objects) GetObjectMetadata(ctx context.Context, repository, ref string, opts *GetObjectMetadataOptions) (*ObjectStats, error) {
	u := fmt.Sprintf("repositories/%s/refs/%s/objects/stat", repository, ref)
	var reply ObjectStats
	_, err := o.client.InvokeWithCredential(ctx, http.MethodGet, u, opts, &reply)
	if err != nil {
		return nil, err
	}
	return &reply, nil
}

type ObjectCopyCreation struct {
	SrcPath string `json:"src_path"`
	SrcRef  string `json:"src_ref"`
	Force   bool   `json:"force"`
	Shallow bool   `json:"shallow"`
}

func (o *Objects) CopyObject(ctx context.Context, repository, branch, destPath string, opts *ObjectCopyCreation) (*ObjectStats, error) {
	u := fmt.Sprintf("repositories/%s/branches/%s/objects/copy?dest_path=%s",
		repository,
		branch,
		url.QueryEscape(destPath),
	)

	var record ObjectStats
	_, err := o.client.InvokeWithCredential(ctx, http.MethodPost, u, opts, &record)
	if err != nil {
		return nil, err
	}
	return &record, err
}

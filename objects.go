package lakefs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
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
	// Range downloads a byte range with a single S3 GetObject request.
	// When Range is set, Concurrency, GetObjectBufferSize, and PartBodyMaxRetries are not used.
	Range       *RangeByteSize `url:"-"`
	IfNoneMatch string         `url:"-"`

	Path string `url:"path,omitempty"`

	// The number of goroutines to spin up in parallel per call to transfer single object parts or directory objects.
	// If this is set to zero, the default transfer concurrency is used.
	//
	// The concurrency pool is not shared between multiple API calls.
	// Default: 5
	// Not used when Range is set.
	Concurrency int

	// Max size for the GetObject memory buffer. The reader returned from GetObject can buffer up to
	// <GetObjectBufferSize> bytes of data at any time and only reads more data when user completely consumes
	// current data buffered. This mechanism avoids unbounded memory usage when downloading large object via GetObject
	// Default: 50 MiB
	// Not used when Range is set.
	GetObjectBufferSize int64

	// PartBodyMaxRetries is the number of retry attempts to make for failed part downloads.
	// Default: 3
	// Not used when Range is set.
	PartBodyMaxRetries int
}

func (o *GetObjectContentOptions) setS3GetOptions(transfer *transfermanager.Options) {
	if o == nil {
		return
	}
	if o.GetObjectBufferSize > 0 {
		transfer.GetObjectBufferSize = o.GetObjectBufferSize
	}
	if o.PartBodyMaxRetries > 0 {
		transfer.PartBodyMaxRetries = o.PartBodyMaxRetries
	}
	if o.Concurrency > 0 {
		transfer.Concurrency = o.Concurrency
	}
}

func (o *GetObjectContentOptions) setS3TransferInput(input *transfermanager.GetObjectInput) {
	if o == nil {
		return
	}
	if o.IfNoneMatch != "" {
		input.IfNoneMatch = aws.String(o.IfNoneMatch)
	}
}

func (o *GetObjectContentOptions) setS3Input(input *s3.GetObjectInput) {
	if o == nil {
		return
	}
	if o.IfNoneMatch != "" {
		input.IfNoneMatch = aws.String(o.IfNoneMatch)
	}
}

func (o *GetObjectContentOptions) Request() []ghttp.RequestFunc {
	if o == nil {
		return nil
	}
	if o.Range == nil && o.IfNoneMatch == "" {
		return nil
	}
	return []ghttp.RequestFunc{
		func(request *http.Request) error {
			if o.IfNoneMatch != "" {
				request.Header.Set("If-None-Match", o.IfNoneMatch)
			}
			if o.Range != nil {
				request.Header.Set("Range", o.Range.String())
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
	if opts == nil {
		return nil, errors.New("get object content options is nil")
	}
	if opts.Path == "" {
		return nil, errors.New("object path is empty")
	}
	s3cc := o.client.s3()
	return s3cc.GetObjectContent(ctx, repository, ref, opts)
}

type DownloadDirectoryOptions struct {
	Path        string
	Destination string

	// IgnoreErrors controls whether DownloadDirectory continues when individual objects fail to download.
	// Default: false
	IgnoreErrors bool

	// Concurrency is the number of goroutines to use for downloading objects.
	// The concurrency pool is not shared between multiple calls.
	// Default: 5
	Concurrency int
}

func (o *DownloadDirectoryOptions) setS3DownloadOptions(transfer *transfermanager.Options) {
	if o == nil {
		return
	}
	transfer.DisableChecksumValidation = true
	if o.Concurrency > 0 {
		transfer.Concurrency = o.Concurrency
	}
}

type DownloadDirectoryStats struct {
	ObjectsDownloaded int64
	ObjectsFailed     int64
}

func (o *Objects) DownloadDirectory(ctx context.Context, repository, ref string, opts *DownloadDirectoryOptions) (*DownloadDirectoryStats, error) {
	if opts == nil {
		return nil, errors.New("download directory options is nil")
	}
	if opts.Destination == "" {
		return nil, errors.New("download directory destination is empty")
	}
	s3cc := o.client.s3()
	return s3cc.DownloadDirectory(ctx, repository, ref, opts)
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

	//Force bool   `url:"force,omitempty"` // force is only supported by lakeFS REST upload API, not S3 gateway upload
	Path string `url:"path,omitempty"`

	// S3 transfer options.
	ContentType string

	// PartSizeBytes is the buffer size, in bytes, used to buffer data into multipart upload parts.
	// The minimum S3 multipart upload part size is 5 MiB.
	// Default: 8 MiB
	PartSizeBytes int64

	// Concurrency is the number of goroutines to use for multipart upload.
	// The concurrency pool is not shared between multiple calls.
	// Default: 5
	Concurrency int

	// MultipartUploadThreshold is the object size threshold, in bytes, for using multipart upload.
	// Default: 16 MiB
	MultipartUploadThreshold int64
}

type UploadDirectoryOptions struct {
	// Source is the local directory to upload.
	Source string

	// Path is the remote directory path under the lakeFS branch.
	Path string

	// Recursive controls whether to upload files in subdirectories.
	// Default: false
	Recursive bool

	// FollowSymbolicLinks controls whether symbolic links are followed while traversing Source.
	// Default: false
	FollowSymbolicLinks bool

	// IgnoreErrors controls whether UploadDirectory continues when individual files fail to upload.
	// Default: false
	IgnoreErrors bool

	// PartSizeBytes is the buffer size, in bytes, used to buffer data into multipart upload parts.
	// The minimum S3 multipart upload part size is 5 MiB.
	// Default: 8 MiB
	PartSizeBytes int64

	// Concurrency is the number of goroutines to use for multipart upload.
	// The concurrency pool is not shared between multiple calls.
	// Default: 5
	Concurrency int

	// MultipartUploadThreshold is the object size threshold, in bytes, for using multipart upload.
	// Default: 16 MiB
	MultipartUploadThreshold int64
}

func (o *UploadDirectoryOptions) setS3UploadOptions(transfer *transfermanager.Options) {
	if o == nil {
		return
	}
	if o.PartSizeBytes > 0 {
		transfer.PartSizeBytes = o.PartSizeBytes
	}
	if o.MultipartUploadThreshold > 0 {
		transfer.MultipartUploadThreshold = o.MultipartUploadThreshold
	}
	if o.Concurrency > 0 {
		transfer.Concurrency = o.Concurrency
	}
}

type UploadDirectoryStats struct {
	ObjectsUploaded int64
	ObjectsFailed   int64
}

func (o *Objects) UploadDirectory(ctx context.Context, repository, branch string, opts *UploadDirectoryOptions) (*UploadDirectoryStats, error) {
	if opts == nil {
		return nil, errors.New("upload directory options is nil")
	}
	if opts.Source == "" {
		return nil, errors.New("upload directory source is empty")
	}
	s3cc := o.client.s3()
	return s3cc.UploadDirectory(ctx, repository, branch, opts)
}

func (c *CreateObjectOptions) contentType(path string) string {
	defaultContentType := "application/octet-stream"
	if c == nil || path == "" {
		return defaultContentType
	}
	mimeType := mime.TypeByExtension(filepath.Ext(path))
	if mimeType == "" {
		return defaultContentType
	}
	return mimeType
}

func (c *CreateObjectOptions) setS3UploadOptions(transfer *transfermanager.Options) {
	if c == nil {
		return
	}
	if c.PartSizeBytes > 0 {
		transfer.PartSizeBytes = c.PartSizeBytes
	}
	if c.MultipartUploadThreshold > 0 {
		transfer.MultipartUploadThreshold = c.MultipartUploadThreshold
	}
	if c.Concurrency > 0 {
		transfer.Concurrency = c.Concurrency
	}
}

func (c *CreateObjectOptions) setS3TransferInput(input *transfermanager.UploadObjectInput) {
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
	if opts == nil {
		return nil, errors.New("create object options is nil")
	}
	if opts.Path == "" {
		return nil, errors.New("object path is empty")
	}

	s3cc := o.client.s3()
	err := s3cc.CreateObject(ctx, repository, branch, reader, opts)
	if err != nil {
		return nil, err
	}

	return o.GetObjectMetadata(ctx, repository, branch, &GetObjectMetadataOptions{
		Path: opts.Path,
	})
}

type DeleteObjectOptions struct {
	Force       bool   `url:"force,omitempty"`
	NoTombstone bool   `url:"no_tombstone,omitempty"`
	Path        string `url:"path,omitempty"`
}

type DeleteDirectoryOptions struct {
	// Path is the remote directory path under the lakeFS branch.
	Path string

	// IgnoreErrors controls whether DeleteDirectory continues reporting success when S3 returns per-object delete errors.
	// Default: false
	IgnoreErrors bool
}

type DeleteDirectoryStats struct {
	ObjectsDeleted int64
	ObjectsFailed  int64
}

func (o *Objects) DeleteDirectory(ctx context.Context, repository, branch string, opts *DeleteDirectoryOptions) (*DeleteDirectoryStats, error) {
	if opts == nil {
		return nil, errors.New("delete directory options is nil")
	}
	if opts.Path == "" {
		return nil, errors.New("delete directory path is empty")
	}
	s3cc := o.client.s3()
	return s3cc.DeleteDirectory(ctx, repository, branch, opts)
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

type RewriteAllObjectMetadataOptions struct {
	Path string             `url:"path,omitempty"`
	Set  ObjectUserMetadata `json:"set,omitempty"`
}

func (o *Objects) RewriteAllObjectMetadata(ctx context.Context, repository, branch string, opts *RewriteAllObjectMetadataOptions) error {
	u := fmt.Sprintf("repositories/%s/branches/%s/objects/stat/user_metadata", repository, branch)
	_, err := o.client.InvokeWithCredential(ctx, http.MethodPut, u, opts, nil, ghttp.Query(opts).Before)
	if err != nil {
		return err
	}
	return nil
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

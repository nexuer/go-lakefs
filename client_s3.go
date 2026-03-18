package lakefs

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/nexuer/ghttp"
)

type s3Client struct {
	cc *s3.Client
}

func newS3Client(credential Credential) (*s3Client, error) {
	if credential == nil {
		return nil, errors.New("credential is nil")
	}
	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithCredentialsProvider(credential.CredentialsProvider()),
		config.WithRegion("us-east-1"),
	)
	if err != nil {
		return nil, err
	}
	cc := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(ghttp.Endpoint(credential.GetEndpoint()))
		o.UsePathStyle = true
		o.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
	})
	return &s3Client{
		cc: cc,
	}, nil
}

type S3UploadOptions struct {
	// The buffer size (in bytes) to use when buffering data into chunks and
	// sending them as parts to S3. The minimum allowed part size is 5MB, and
	// if this value is set to zero, the DefaultUploadPartSize value will be used.
	// Default: 1024 * 1024 * 8
	PartSizeBytes int64

	// The number of goroutines to spin up in parallel per call to transfer single object parts or directory objects.
	// If this is set to zero, the DefaultUploadConcurrency value will be used.
	//
	// The concurrency pool is not shared between multiple API calls.
	// Default: 5
	Concurrency int

	// The threshold bytes to decide when the file should be multi-uploaded
	// Default: 1024 * 1024 * 16
	MultipartUploadThreshold int64
}

func (o *S3UploadOptions) set(transfer *transfermanager.Options) {
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

// CreateObject 支持分片上传、多协程上传
func (s *s3Client) CreateObject(ctx context.Context, repository, branch string, reader io.Reader, opts *CreateObjectOptions) error {
	transfer := transfermanager.New(s.cc, func(o *transfermanager.Options) {
		opts.setS3UploadOptions(o)
	})
	input := &transfermanager.UploadObjectInput{
		Bucket: aws.String(repository),
		Key:    aws.String(branch + "/" + strings.TrimLeft(opts.Path, "/")),
		Body:   reader,
	}

	opts.setS3TransferInput(input)

	_, err := transfer.UploadObject(ctx, input)

	return err
}

type S3GetObjectOptions struct {
	// The number of goroutines to spin up in parallel per call to transfer single object parts or directory objects.
	// If this is set to zero, the DefaultUploadConcurrency value will be used.
	//
	// The concurrency pool is not shared between multiple API calls.
	// Default: 5
	Concurrency int

	// Max size for the GetObject memory buffer. The reader returned from GetObject can buffer up to
	// <GetObjectBufferSize> bytes of data at any time and only reads more data when user completely consumes
	// current data buffered. This mechanism avoids unbounded memory usage when downloading large object via GetObject
	// Default: 1024 * 1024 * 50
	GetObjectBufferSize int64

	// PartBodyMaxRetries is the number of retry attempts to make for failed part downloads.
	// Default: 3
	PartBodyMaxRetries int
}

func (o *S3GetObjectOptions) set(transfer *transfermanager.Options) {
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

func (s *s3Client) GetObjectContent(ctx context.Context, repository, ref string, opts *GetObjectContentOptions) (*ObjectContent, error) {
	bucket := aws.String(repository)
	key := aws.String(ref + "/" + strings.TrimLeft(opts.Path, "/"))
	if opts.Range != nil {
		input := &s3.GetObjectInput{
			Bucket: bucket,
			Key:    key,
			Range:  aws.String(opts.Range.String()),
		}
		opts.setS3Input(input)
		out, err := s.cc.GetObject(ctx, input)
		if err != nil {
			return nil, err
		}
		return &ObjectContent{
			Headers: ObjectHeaders{
				ETag:          aws.ToString(out.ETag),
				LastModified:  aws.ToTime(out.LastModified),
				ContentLength: aws.ToInt64(out.ContentLength),
				ContentRange:  aws.ToString(out.ContentRange),
			},
			Body: out.Body,
		}, nil
	}
	// 全量文件进行分块
	transfer := transfermanager.New(s.cc, func(o *transfermanager.Options) {
		opts.setS3GetOptions(o)
		o.DisableChecksumValidation = true
		o.GetObjectType = types.GetObjectRanges
	})

	input := &transfermanager.GetObjectInput{
		Bucket: bucket,
		Key:    key,
	}
	opts.setS3TransferInput(input)
	out, err := transfer.GetObject(ctx, input)
	if err != nil {
		return nil, err
	}
	return &ObjectContent{
		Headers: ObjectHeaders{
			ETag:          aws.ToString(out.ETag),
			LastModified:  aws.ToTime(out.LastModified),
			ContentLength: aws.ToInt64(out.ContentLength),
			ContentRange:  aws.ToString(out.ContentRange),
		},
		Body: io.NopCloser(out.Body),
	}, nil
}

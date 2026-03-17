package lakefs

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
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

type TransferOptions struct {
	// The buffer size (in bytes) to use when buffering data into chunks and
	// sending them as parts to S3. The minimum allowed part size is 5MB, and
	// if this value is set to zero, the DefaultUploadPartSize value will be used.
	PartSizeBytes int64

	// The number of goroutines to spin up in parallel per call to transfer single object parts or directory objects.
	// If this is set to zero, the DefaultUploadConcurrency value will be used.
	//
	// The concurrency pool is not shared between multiple API calls.
	Concurrency int
}

func (t *TransferOptions) set(opts *transfermanager.Options) {
	if t == nil {
		return
	}
	opts.PartSizeBytes = t.PartSizeBytes
	opts.Concurrency = t.Concurrency
}

// CreateObject 支持分片上传、多协程上传
func (s *s3Client) CreateObject(ctx context.Context, repository, branch string, reader io.Reader, opts *CreateObjectOptions) error {
	transfer := transfermanager.New(s.cc)
	input := &transfermanager.UploadObjectInput{
		Bucket: aws.String(repository),
		Key:    aws.String(branch + "/" + strings.TrimLeft(opts.Path, "/")),
		Body:   reader,
	}

	opts.setS3Options(input)

	_, err := transfer.UploadObject(ctx, input, func(options *transfermanager.Options) {
		opts.Transfer.set(options)
	})

	return err
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
		opts.setS3Options0(input)
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
	transfer := transfermanager.New(s.cc)
	input := &transfermanager.GetObjectInput{
		Bucket: bucket,
		Key:    key,
	}
	opts.setS3Options(input)
	out, err := transfer.GetObject(ctx, input, func(options *transfermanager.Options) {
		opts.Transfer.set(options)
		options.DisableChecksumValidation = true
	})
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

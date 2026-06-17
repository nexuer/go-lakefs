package lakefs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
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

func lakeFSS3Key(ref, path string) string {
	path = strings.TrimLeft(path, "/")
	if path == "" {
		return ref
	}
	return ref + "/" + path
}

func lakeFSS3Prefix(ref, path string) string {
	ref = strings.Trim(ref, "/")
	path = strings.Trim(path, "/")
	if path == "" {
		return ref + "/"
	}

	return ref + "/" + path + "/"
}

// CreateObject 支持分片上传、多协程上传
func (s *s3Client) CreateObject(ctx context.Context, repository, branch string, reader io.Reader, opts *CreateObjectOptions) error {
	transfer := transfermanager.New(s.cc, func(o *transfermanager.Options) {
		opts.setS3UploadOptions(o)
	})

	input := &transfermanager.UploadObjectInput{
		Bucket:      aws.String(repository),
		Key:         aws.String(lakeFSS3Key(branch, opts.Path)),
		Body:        reader,
		ContentType: aws.String(opts.contentType(opts.Path)),
	}

	opts.setS3TransferInput(input)

	_, err := transfer.UploadObject(ctx, input)

	return err
}

func (s *s3Client) UploadDirectory(ctx context.Context, repository, branch string, opts *UploadDirectoryOptions) (*UploadDirectoryStats, error) {
	transfer := transfermanager.New(s.cc, func(o *transfermanager.Options) {
		opts.setS3UploadOptions(o)
	})

	failurePolicy := transfermanager.UploadDirectoryFailurePolicy(transfermanager.TerminateUploadPolicy{})
	if opts.IgnoreErrors {
		failurePolicy = transfermanager.IgnoreUploadFailurePolicy{}
	}
	input := &transfermanager.UploadDirectoryInput{
		Bucket:              aws.String(repository),
		Source:              aws.String(opts.Source),
		KeyPrefix:           aws.String(lakeFSS3Key(branch, opts.Path)),
		Recursive:           aws.Bool(opts.Recursive),
		FollowSymbolicLinks: aws.Bool(opts.FollowSymbolicLinks),
		FailurePolicy:       failurePolicy,
	}

	out, err := transfer.UploadDirectory(ctx, input)
	if err != nil {
		return nil, err
	}

	return &UploadDirectoryStats{
		ObjectsUploaded: out.ObjectsUploaded,
		ObjectsFailed:   out.ObjectsFailed,
	}, nil
}

func (s *s3Client) DeleteDirectory(ctx context.Context, repository, branch string, opts *DeleteDirectoryOptions) (*DeleteDirectoryStats, error) {
	bucket := aws.String(repository)
	prefix := aws.String(lakeFSS3Prefix(branch, opts.Path))
	var objectsDeleted int64
	var continuationToken *string

	for {
		listOut, err := s.cc.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            bucket,
			Prefix:            prefix,
			ContinuationToken: continuationToken,
		})
		if err != nil {
			return nil, err
		}

		objects := make([]s3types.ObjectIdentifier, 0, len(listOut.Contents))
		for _, object := range listOut.Contents {
			if object.Key == nil {
				continue
			}
			objects = append(objects, s3types.ObjectIdentifier{Key: object.Key})
		}

		for len(objects) > 0 {
			batchSize := len(objects)
			if batchSize > 1000 {
				batchSize = 1000
			}
			deleteOut, err := s.cc.DeleteObjects(ctx, &s3.DeleteObjectsInput{
				Bucket: bucket,
				Delete: &s3types.Delete{
					Objects: objects[:batchSize],
					Quiet:   aws.Bool(true),
				},
			})
			if err != nil {
				return nil, err
			}
			objectsDeleted += int64(batchSize - len(deleteOut.Errors))
			if len(deleteOut.Errors) > 0 && !opts.IgnoreErrors {
				firstErr := deleteOut.Errors[0]
				return &DeleteDirectoryStats{
					ObjectsDeleted: objectsDeleted,
					ObjectsFailed:  int64(len(deleteOut.Errors)),
				}, fmt.Errorf("delete directory failed for %d objects, first key %q: %s", len(deleteOut.Errors), aws.ToString(firstErr.Key), aws.ToString(firstErr.Message))
			}
			objects = objects[batchSize:]
		}

		if !aws.ToBool(listOut.IsTruncated) {
			break
		}
		continuationToken = listOut.NextContinuationToken
	}

	return &DeleteDirectoryStats{ObjectsDeleted: objectsDeleted}, nil
}

func (s *s3Client) DownloadDirectory(ctx context.Context, repository, ref string, opts *DownloadDirectoryOptions) (*DownloadDirectoryStats, error) {
	transfer := transfermanager.New(s.cc, func(o *transfermanager.Options) {
		opts.setS3DownloadOptions(o)
	})

	failurePolicy := transfermanager.DownloadDirectoryFailurePolicy(transfermanager.TerminateDownloadPolicy{})
	if opts.IgnoreErrors {
		failurePolicy = transfermanager.IgnoreDownloadFailurePolicy{}
	}
	out, err := transfer.DownloadDirectory(ctx, &transfermanager.DownloadDirectoryInput{
		Bucket:        aws.String(repository),
		Destination:   aws.String(opts.Destination),
		KeyPrefix:     aws.String(lakeFSS3Prefix(ref, opts.Path)),
		FailurePolicy: failurePolicy,
	})
	if err != nil {
		return nil, err
	}
	stats := &DownloadDirectoryStats{
		ObjectsDownloaded: out.ObjectsDownloaded,
		ObjectsFailed:     out.ObjectsFailed,
	}
	if stats.ObjectsDownloaded == 0 && stats.ObjectsFailed == 0 {
		return stats, newStatusCode(http.StatusNotFound, fmt.Errorf("directory %q not found", opts.Path))
	}
	return stats, nil
}

func (s *s3Client) GetObjectContent(ctx context.Context, repository, ref string, opts *GetObjectContentOptions) (*ObjectContent, error) {
	bucket := aws.String(repository)
	key := aws.String(lakeFSS3Key(ref, opts.Path))
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

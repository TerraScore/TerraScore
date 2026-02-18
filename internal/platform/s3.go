package platform

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Client wraps an S3 client for presigning and direct object operations.
type S3Client struct {
	client    *s3.Client
	presigner *s3.PresignClient
	bucket    string
}

// NewS3Client creates an S3 presign client.
// Supports AWS S3, Hetzner Object Storage, and MinIO via custom endpoint.
func NewS3Client(cfg AWSConfig) (*S3Client, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	var opts []func(*s3.Options)
	if cfg.S3Endpoint != "" {
		opts = append(opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.S3Endpoint)
			o.UsePathStyle = true // Required for MinIO and Hetzner
		})
	}

	client := s3.NewFromConfig(awsCfg, opts...)
	presigner := s3.NewPresignClient(client)

	return &S3Client{
		client:    client,
		presigner: presigner,
		bucket:    cfg.S3Bucket,
	}, nil
}

// GeneratePresignedPutURL generates a presigned PUT URL for uploading to S3.
// Key pattern: media/{job_id}/{step_id}/{uuid}.{ext}
func (c *S3Client) GeneratePresignedPutURL(ctx context.Context, key, contentType string, ttl time.Duration) (string, error) {
	if ttl == 0 {
		ttl = 15 * time.Minute
	}

	req, err := c.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("presigning PUT URL: %w", err)
	}

	return req.URL, nil
}

// GeneratePresignedGetURL generates a presigned GET URL for downloading from S3.
func (c *S3Client) GeneratePresignedGetURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if ttl == 0 {
		ttl = 15 * time.Minute
	}

	req, err := c.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("presigning GET URL: %w", err)
	}

	return req.URL, nil
}

// PutObject uploads an object to S3.
func (c *S3Client) PutObject(ctx context.Context, key, contentType string, body io.Reader) error {
	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
		Body:        body,
	})
	if err != nil {
		return fmt.Errorf("uploading object to S3: %w", err)
	}
	return nil
}

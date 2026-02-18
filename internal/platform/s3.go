package platform

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Client wraps an S3 presign client for generating presigned URLs.
type S3Client struct {
	presigner *s3.PresignClient
	bucket    string
}

// NewS3Client creates an S3 presign client using default AWS credentials.
func NewS3Client(cfg AWSConfig) (*S3Client, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg)
	presigner := s3.NewPresignClient(client)

	return &S3Client{
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

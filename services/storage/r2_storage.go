package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2Storage struct {
	client     *s3.Client
	bucketName string
}

func NewR2Storage() (*R2Storage, error) {
	r2AccountId := os.Getenv("R2_ACCOUNT_ID")
	r2AccessKeyId := os.Getenv("R2_ACCESS_KEY_ID")
	r2AccessKeySecret := os.Getenv("R2_ACCESS_KEY_SECRET")
	bucketName := os.Getenv("R2_BUCKET_NAME")

	if r2AccountId == "" || r2AccessKeyId == "" || r2AccessKeySecret == "" || bucketName == "" {
		return nil, fmt.Errorf("missing required R2 configuration")
	}

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", r2AccountId),
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     r2AccessKeyId,
				SecretAccessKey: r2AccessKeySecret,
			}, nil
		})),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load R2 config: %w", err)
	}

	client := s3.NewFromConfig(cfg)
	return &R2Storage{
		client:     client,
		bucketName: bucketName,
	}, nil
}

func (r *R2Storage) Save(ctx context.Context, file io.Reader, userID uint, fileName string) error {
	key := fmt.Sprintf("user_%d/%s", userID, fileName)

	metadata := map[string]string{
		"user-id":     fmt.Sprintf("%d", userID),
		"upload-time": time.Now().UTC().Format(time.RFC3339),
	}

	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:   aws.String(r.bucketName),
		Key:      aws.String(key),
		Body:     file,
		Metadata: metadata,
	})

	if err != nil {
		return fmt.Errorf("failed to upload to R2: %w", err)
	}

	return nil
}

func (r *R2Storage) GetTotalUsage(ctx context.Context, userID uint) (int64, error) {
	var total int64
	prefix := fmt.Sprintf("user_%d/", userID)

	paginator := s3.NewListObjectsV2Paginator(r.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(r.bucketName),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return 0, fmt.Errorf("failed to list R2 objects: %w", err)
		}

		for _, obj := range page.Contents {
			if obj.Size != nil {
				total += *obj.Size
			}
		}
	}

	return total, nil
}

func (r *R2Storage) Delete(ctx context.Context, userID uint, fileName string) error {
	key := fmt.Sprintf("user_%d/%s", userID, fileName)

	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("failed to delete from R2: %w", err)
	}

	return nil
}

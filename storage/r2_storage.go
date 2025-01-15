package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/sirupsen/logrus"
)

type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
}

type R2Storage struct {
	Client *s3.Client
	Bucket string
}

func NewR2Storage(cfg R2Config) (*R2Storage, error) {
	endpointResolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:           "https://" + cfg.AccountID + ".r2.cloudflarestorage.com",
			SigningRegion: "auto",
		}, nil
	})

	creds := aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
		return aws.Credentials{
			AccessKeyID:     cfg.AccessKeyID,
			SecretAccessKey: cfg.SecretAccessKey,
		}, nil
	})

	opts := []func(*config.LoadOptions) error{
		config.WithRegion("auto"),
		config.WithEndpointResolver(endpointResolver),
		config.WithCredentialsProvider(creds),
	}

	ctx := context.TODO()
	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		logrus.Errorf("Failed to load Cloudflare R2 configuration: %v", err)
		return nil, err
	}

	client := s3.NewFromConfig(awsCfg)
	logrus.Info("Successfully configured R2 storage")
	return &R2Storage{Client: client, Bucket: cfg.Bucket}, nil
}

// Download downloads a file from R2 storage
func (s *R2Storage) Download(filename string) (*os.File, error) {
	logrus.WithFields(logrus.Fields{
		"filename": filename,
		"bucket":   s.Bucket,
	}).Info("Initiating file download")

	tmpFile, err := os.CreateTemp("", "download-*")
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"filename": filename,
			"error":    err,
		}).Error("Error creating temporary file")
		return nil, err
	}
	defer func() {
		if err != nil {
			tmpFile.Close()
		}
	}()

	downloadResult, err := s.Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(filename),
	})
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"filename": filename,
			"error":    err,
		}).Error("Error downloading file")
		return nil, err
	}
	defer downloadResult.Body.Close()

	if _, err := io.Copy(tmpFile, downloadResult.Body); err != nil {
		logrus.WithFields(logrus.Fields{
			"filename": filename,
			"error":    err,
		}).Error("Error writing file to disk")
		return nil, err
	}
	logrus.WithFields(logrus.Fields{
		"filename": filename,
	}).Info("Downloaded file successfully")

	return tmpFile, nil
}

// Upload uploads a file to R2 storage
func (s *R2Storage) Upload(file io.Reader, filename string) (string, error) {
	logrus.WithFields(logrus.Fields{
		"filename": filename,
		"bucket":   s.Bucket,
	}).Info("Initiating file upload")

	_, err := s.Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(filename),
		Body:   file,
	})
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"filename": filename,
			"error":    err,
		}).Error("Error uploading file")
		return "", err
	}

	logrus.WithFields(logrus.Fields{
		"filename": filename,
	}).Info("Uploaded file successfully")

	return filename, nil
}

// Delete removes a file from R2 storage
func (s *R2Storage) Delete(filename string) error {
	logrus.WithFields(logrus.Fields{
		"filename": filename,
		"bucket":   s.Bucket,
	}).Info("Initiating file deletion")

	_, err := s.Client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(filename),
	})
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"filename": filename,
			"error":    err,
		}).Error("Error deleting file")
		return err
	}

	logrus.WithFields(logrus.Fields{
		"filename": filename,
	}).Info("Deleted file successfully")

	return nil
}

// Exists checks if a file exists in R2 storage
func (s *R2Storage) Exists(filePath string) (bool, error) {
	_, err := s.Client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(filePath),
	})
	var notFound *types.NoSuchKey
	if errors.As(err, &notFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c R2Config) Validate() error {
	if c.AccountID == "" {
		return fmt.Errorf("account ID is required")
	}
	if c.AccessKeyID == "" {
		return fmt.Errorf("access key ID is required")
	}
	if c.SecretAccessKey == "" {
		return fmt.Errorf("secret access key is required")
	}
	if c.Bucket == "" {
		return fmt.Errorf("bucket name is required")
	}
	return nil
}

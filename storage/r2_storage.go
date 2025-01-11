package storage

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/sirupsen/logrus"
)

type R2Storage struct {
	Client *s3.Client
	Bucket string
}

// NewR2Storage creates a new R2Storage instance
func NewR2Storage(bucket string) (*R2Storage, error) {
	// Configuration for Cloudflare R2
	configLoadOptions := []func(*config.LoadOptions) error{
		config.WithRegion("auto"),
	}
	// Configure the resolver endpoint for Cloudflare R2
	endpointResolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:           "https://" + os.Getenv("R2_ACCOUNT_ID") + ".r2.cloudflarestorage.com",
			SigningRegion: "auto",
		}, nil
	})

	// Load the configuration with the defined options
	ctx := context.TODO()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	opts := append(configLoadOptions, config.WithEndpointResolver(endpointResolver))

	// Configure credentials for R2
	creds := aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
		return aws.Credentials{
			AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
		}, nil
	})
	opts = append(opts, config.WithCredentialsProvider(creds))

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		logrus.Errorf("Failed to load Cloudflare R2 configuration: %v", err)
		return nil, err
	}

	client := s3.NewFromConfig(cfg)
	logrus.Info("Successfully configured R2 storage")
	return &R2Storage{Client: client, Bucket: bucket}, nil
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

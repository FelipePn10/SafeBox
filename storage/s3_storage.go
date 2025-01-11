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

type S3Storage struct {
	Client *s3.Client
	Bucket string
}

func NewS3Storage(bucket string) (*S3Storage, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("auto"),
		config.WithEndpointResolver(aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:           "https://" + os.Getenv("R2_ACCOUNT_ID") + ".r2.cloudflarestorage.com",
				SigningRegion: "auto",
			}, nil
		})),
		config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
				SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
			}, nil
		})),
	)
	if err != nil {
		logrus.Errorf("Failed to load AWS configuration: %v", err)
		return nil, err
	}

	client := s3.NewFromConfig(cfg)
	logrus.Info("Successfully configured S3 storage")
	return &S3Storage{Client: client, Bucket: bucket}, nil
}

func (s *S3Storage) Download(filename string) (*os.File, error) {
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

func (s *S3Storage) Upload(file io.Reader, filename string) (string, error) {
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

func (s *S3Storage) Delete(filename string) error {
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

// Exists checks if a file exists in the S3 storage
func (s *S3Storage) Exists(filePath string) (bool, error) {
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

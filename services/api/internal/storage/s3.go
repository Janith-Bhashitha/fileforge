package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

// S3Store is the Phase 6 backend. It satisfies the same Store interface as
// LocalStore, so no handler, worker or processor changes when the storage
// backend does — swapping them is a config change, which is the whole point
// of having kept storage keys opaque since Phase 2.
//
// It also drives MinIO (and LocalStack) unchanged: those speak the S3 API,
// so the same code is testable locally without an AWS account.
type S3Store struct {
	client  *s3.Client
	presign *s3.PresignClient
	bucket  string
	workDir string
}

type S3Config struct {
	Bucket string
	Region string

	// Endpoint and ForcePathStyle exist for MinIO/LocalStack. Left empty,
	// the SDK talks to real AWS with virtual-host addressing.
	Endpoint       string
	ForcePathStyle bool

	// PublicEndpoint is the address a *browser* can reach, when that
	// differs from the one the server uses. Presigned URLs are handed to
	// the client, so signing them against an internal hostname (a Docker
	// service name, a VPC endpoint) produces links nobody outside can
	// open. Empty means the two are the same, which is the normal AWS case.
	PublicEndpoint string

	// Static credentials, for MinIO or an explicitly-configured deployment.
	// Left empty, the SDK's default chain applies (env, shared config, and
	// on ECS the task role — which is what production should use).
	AccessKeyID     string
	SecretAccessKey string

	// WorkDir is scratch space for files pulled down to be processed.
	WorkDir string
}

func NewS3Store(ctx context.Context, cfg S3Config) (*S3Store, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("bucket is required")
	}
	if err := os.MkdirAll(cfg.WorkDir, 0o755); err != nil {
		return nil, fmt.Errorf("create work dir: %w", err)
	}

	loadOpts := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(cfg.Region)}
	if cfg.AccessKeyID != "" {
		loadOpts = append(loadOpts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.UsePathStyle = cfg.ForcePathStyle
	})

	// Presigning goes through its own client so the signature is computed
	// against the host the client will actually call. A URL signed for one
	// hostname does not verify when requested at another.
	presignSource := client
	if cfg.PublicEndpoint != "" {
		presignSource = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.PublicEndpoint)
			o.UsePathStyle = cfg.ForcePathStyle
		})
	}

	return &S3Store{
		client:  client,
		presign: s3.NewPresignClient(presignSource),
		bucket:  cfg.Bucket,
		workDir: cfg.WorkDir,
	}, nil
}

// Keys stay fully opaque — a UUID plus extension, never the user's filename.
// The original name lives in the database, so an object key leaks nothing
// and can't be guessed or enumerated.
func (s *S3Store) Save(ctx context.Context, data []byte, ext string) (string, error) {
	key := uuid.New().String() + ext
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return "", fmt.Errorf("put object: %w", err)
	}
	return key, nil
}

// SaveFile takes ownership of localPath: once the object is safely in the
// bucket the scratch copy is removed, since nothing reads from local disk
// under this backend. (LocalStore's SaveFile keeps the file, because there
// the file *is* the stored object.)
func (s *S3Store) SaveFile(ctx context.Context, localPath string) (string, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("open produced file: %w", err)
	}

	key := uuid.New().String() + filepath.Ext(localPath)
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   f,
	})
	f.Close()
	if err != nil {
		return "", fmt.Errorf("put object: %w", err)
	}

	os.Remove(localPath)
	return key, nil
}

// Fetch downloads an object into WorkDir and returns the cleanup that
// deletes it. Callers must defer the cleanup: scratch files are the one
// thing that silently fills a disk.
func (s *S3Store) Fetch(ctx context.Context, key string) (string, func(), error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", func() {}, fmt.Errorf("get object: %w", err)
	}
	defer out.Body.Close()

	localPath := filepath.Join(s.workDir, key)
	f, err := os.Create(localPath)
	if err != nil {
		return "", func() {}, fmt.Errorf("create scratch file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, out.Body); err != nil {
		os.Remove(localPath)
		return "", func() {}, fmt.Errorf("download object: %w", err)
	}

	return localPath, func() { os.Remove(localPath) }, nil
}

func (s *S3Store) WorkDir() string { return s.workDir }

func (s *S3Store) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

// PresignPut gives the browser a short-lived URL to upload straight to S3,
// so file bytes never pass through the API at all. PresignGet does the same
// for downloads.
func (s *S3Store) PresignPut(ctx context.Context, ext string, ttl time.Duration) (url, key string, err error) {
	key = uuid.New().String() + ext
	req, err := s.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", "", fmt.Errorf("presign put: %w", err)
	}
	return req.URL, key, nil
}

func (s *S3Store) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	req, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("presign get: %w", err)
	}
	return req.URL, nil
}

// Exists is how the API confirms a presigned upload actually landed before
// creating a job that references it — the browser's success isn't proof.
func (s *S3Store) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

package product

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2Objects struct {
	bucket string
	client *s3.Client
}

func NewR2Objects(ctx context.Context) (*R2Objects, error) {
	endpoint, account, accessKey, secret, bucket := os.Getenv("R2_ENDPOINT"), os.Getenv("R2_ACCOUNT_ID"), os.Getenv("R2_ACCESS_KEY_ID"), os.Getenv("R2_SECRET_ACCESS_KEY"), os.Getenv("R2_BUCKET")
	if endpoint == "" && account != "" {
		endpoint = fmt.Sprintf("https://%s.r2.cloudflarestorage.com", account)
	}
	if endpoint == "" || accessKey == "" || secret == "" || bucket == "" {
		return nil, fmt.Errorf("R2_ENDPOINT (or R2_ACCOUNT_ID), R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY, and R2_BUCKET are required")
	}
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("auto"), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secret, "")), config.WithBaseEndpoint(endpoint))
	if err != nil {
		return nil, err
	}
	return &R2Objects{bucket: bucket, client: s3.NewFromConfig(cfg, func(o *s3.Options) { o.UsePathStyle = true })}, nil
}
func (o *R2Objects) Put(ctx context.Context, key, typ string, value []byte) error {
	_, err := o.client.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(o.bucket), Key: aws.String(key), Body: bytes.NewReader(value), ContentType: aws.String(typ), CacheControl: aws.String("private, max-age=0, no-store")})
	return err
}
func (o *R2Objects) Get(ctx context.Context, key string) ([]byte, string, error) {
	result, err := o.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(o.bucket), Key: aws.String(key)})
	if err != nil {
		return nil, "", err
	}
	defer result.Body.Close()
	b := new(bytes.Buffer)
	if _, err = b.ReadFrom(result.Body); err != nil {
		return nil, "", err
	}
	typ := contentType(key)
	if result.ContentType != nil && *result.ContentType != "" {
		typ = *result.ContentType
	}
	return b.Bytes(), typ, nil
}
func (o *R2Objects) Delete(ctx context.Context, key string) error {
	_, err := o.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(o.bucket), Key: aws.String(key)})
	return err
}

func R2Enabled() bool { return strings.TrimSpace(os.Getenv("R2_BUCKET")) != "" }

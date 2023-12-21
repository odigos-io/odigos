package awss3exporter

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.uber.org/zap"
	"math/rand"
	"strconv"
	"time"
)

type S3Writer struct {
	s3Client *s3.Client
}

// generate the s3 time key based on partition configuration
func getTimeKey(time time.Time, partition string) string {
	var timeKey string
	year, month, day := time.Date()
	hour, minute, _ := time.Clock()

	if partition == "hour" {
		timeKey = fmt.Sprintf("year=%d/month=%02d/day=%02d/hour=%02d", year, month, day, hour)
	} else {
		timeKey = fmt.Sprintf("year=%d/month=%02d/day=%02d/hour=%02d/minute=%02d", year, month, day, hour, minute)
	}
	return timeKey
}

func randomInRange(low, hi int) int {
	return low + rand.Intn(hi-low)
}

func getS3Key(time time.Time, keyPrefix string, partition string, filePrefix string, metadata string, fileformat string) string {
	timeKey := getTimeKey(time, partition)
	randomID := randomInRange(100000000, 999999999)

	var s3Key string
	if keyPrefix != "" {
		s3Key += keyPrefix + "/"
	}

	s3Key += timeKey + "/" + filePrefix + metadata + "_" + strconv.Itoa(randomID) + "." + fileformat

	return s3Key
}

func (s3Writer *S3Writer) WriteBuffer(ctx context.Context, buf []byte, config *Config, metadata string, format string) error {
	now := time.Now()
	key := getS3Key(now,
		config.AWSS3UploadConfig.S3Prefix, config.AWSS3UploadConfig.S3Partition,
		config.AWSS3UploadConfig.FilePrefix, metadata, format)

	config.logger.Info("Writing to S3", zap.String("key", key))

	// Write to S3
	_, err := s3Writer.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(config.AWSS3UploadConfig.S3Bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(buf),
	})
	return err
}

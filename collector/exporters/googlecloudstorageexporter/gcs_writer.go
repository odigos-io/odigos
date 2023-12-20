package googlecloudstorageexporter

import (
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"go.uber.org/zap"
	"math/rand"
	"strconv"
	"time"
)

type GCSWriter struct {
	gcsClient *storage.Client
}

// generate the gcs time key based on partition configuration
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

func getGCSKey(time time.Time, keyPrefix string, partition string, filePrefix string, metadata string, fileformat string) string {
	timeKey := getTimeKey(time, partition)
	randomID := randomInRange(100000000, 999999999)

	gcsKey := keyPrefix + "/" + timeKey + "/" + filePrefix + metadata + "_" + strconv.Itoa(randomID) + "." + fileformat

	return gcsKey
}

func (gcsWriter *GCSWriter) WriteBuffer(ctx context.Context, buf []byte, config *Config, metadata string, format string) error {
	now := time.Now()
	key := getGCSKey(now,
		config.GCSUploader.GCSPrefix, config.GCSUploader.GCSPartition,
		config.GCSUploader.FilePrefix, metadata, format)

	config.logger.Info("Writing to GCS", zap.String("key", key))

	// Write to GCS
	bucket := gcsWriter.gcsClient.Bucket(config.GCSUploader.GCSBucket)
	obj := bucket.Object(key)
	w := obj.NewWriter(ctx)

	// write the buffer to GCS
	_, err := w.Write(buf)
	if err != nil {
		return err
	}

	// close the writer
	err = w.Close()
	if err != nil {
		return err
	}

	// create a reader from data data in memory
	//reader := bytes.NewReader(buf)
	//
	//sess, err := session.NewSession(&aws.Config{
	//	Region: aws.String(config.S3Uploader.Region)},
	//)
	//
	//if err != nil {
	//	return err
	//}
	//
	//uploader := s3manager.NewUploader(sess)
	//
	//_, err = uploader.Upload(&s3manager.UploadInput{
	//	Bucket: aws.String(config.S3Uploader.S3Bucket),
	//	Key:    aws.String(key),
	//	Body:   reader,
	//})
	//if err != nil {
	//	return err
	//}

	return nil
}

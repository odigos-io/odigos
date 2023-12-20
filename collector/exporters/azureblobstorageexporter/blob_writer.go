package azureblobstorageexporter

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"go.uber.org/zap"
	"math/rand"
	"strconv"
	"time"
)

type ABSWriter struct {
	azureClient *azblob.Client
}

// generate the azure blob time key based on partition configuration
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

func getAzureBlobKey(time time.Time, partition string, filePrefix string, metadata string, fileformat string) string {
	timeKey := getTimeKey(time, partition)
	randomID := randomInRange(100000000, 999999999)

	key := timeKey + "/" + filePrefix + metadata + "_" + strconv.Itoa(randomID) + "." + fileformat

	return key
}

func (absWriter *ABSWriter) WriteBuffer(ctx context.Context, buf []byte, config *Config, metadata string, format string) error {
	now := time.Now()
	key := getAzureBlobKey(now, config.ABSUploader.ABSPartition,
		config.ABSUploader.FilePrefix, metadata, format)

	config.logger.Info("Writing to Azure Blob Storage", zap.String("key", key))

	// Write to Azure Blob storage
	_, err := absWriter.azureClient.UploadBuffer(ctx, config.ABSUploader.ABSContainer, key, buf, nil)
	if err != nil {
		return err
	}

	return nil
}

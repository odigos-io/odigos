package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/logicmonitor/lm-data-sdk-go/api/logs"
	"github.com/logicmonitor/lm-data-sdk-go/model"
	"github.com/logicmonitor/lm-data-sdk-go/utils"
	"github.com/logicmonitor/lm-data-sdk-go/utils/translator"
)

func main() {
	logMessage := "This is a test message"

	options := []logs.Option{
		logs.WithLogBatchingDisabled(),
		logs.WithRateLimit(2),
	}

	lmLog, err := logs.NewLMLogIngest(context.Background(), options...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when initializing log client: %v", err)
		return
	}

	resourceIDs := map[string]interface{}{"system.displayname": "example-cart-service"}
	metadata := map[string]interface{}{"testKey": "testValue"}

	fmt.Println("Sending log1....")
	logInput := translator.ConvertToLMLogInput(logMessage, utils.NewTimestampFromTime(time.Now()).String(), resourceIDs, metadata)
	_, err = lmLog.SendLogs(context.Background(), []model.LogInput{logInput})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error in sending log: %v", err)
	}
	time.Sleep(10 * time.Second)
}

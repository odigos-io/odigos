package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var ingestURL = "https://%s.logicmonitor.com/rest"

const REGEX_COMPANY_NAME = "^[a-zA-Z0-9_.\\-]+$"

type Lmv1Token struct {
	AccessID  string
	Signature string
	Epoch     time.Time
}

type AuthParams struct {
	AccessID  string
	AccessKey string

	BearerToken string

	CollectorCredentialsProvider func() string
}

func (ap AuthParams) GetCredentials(method, resourcePath string, body []byte) (string, error) {
	accessID := ap.AccessID
	if accessID == "" {
		accessID = os.Getenv("LOGICMONITOR_ACCESS_ID")
	}
	accessKey := ap.AccessKey
	if accessKey == "" {
		accessKey = os.Getenv("LOGICMONITOR_ACCESS_KEY")
	}
	bearerToken := ap.BearerToken
	if bearerToken == "" {
		bearerToken = os.Getenv("LOGICMONITOR_BEARER_TOKEN")
	}

	if ap.CollectorCredentialsProvider != nil {
		if credentials := ap.CollectorCredentialsProvider(); credentials != "" {
			return credentials, nil
		}
	} else if accessID != "" && accessKey != "" {
		return generateLMv1Token(method, accessID, accessKey, body, resourcePath).String(), nil
	} else if bearerToken != "" {
		return bearerToken, nil
	}
	return "", errors.New("GetCredentials: auth token not found")
}

func (t *Lmv1Token) String() string {
	builder := strings.Builder{}
	append := func(s string) {
		if _, err := builder.WriteString(s); err != nil {
			panic(err)
		}
	}
	append("LMv1 ")
	append(t.AccessID)
	append(":")
	append(t.Signature)
	append(":")
	append(strconv.FormatInt(t.Epoch.UnixNano()/1000000, 10))

	return builder.String()
}

// generateLMv1Token generate LMv1Token
func generateLMv1Token(method string, accessID string, accessKey string, body []byte, resourcePath string) *Lmv1Token {

	epochTime := time.Now()
	epoch := strconv.FormatInt(epochTime.UnixNano()/1000000, 10)

	methodUpper := strings.ToUpper(method)

	h := hmac.New(sha256.New, []byte(accessKey))

	writeOrPanic := func(bs []byte) {
		if _, err := h.Write(bs); err != nil {
			panic(err)
		}
	}
	writeOrPanic([]byte(methodUpper))
	writeOrPanic([]byte(epoch))
	if body != nil {
		writeOrPanic(body)
	}
	writeOrPanic([]byte(resourcePath))

	hash := h.Sum(nil)
	hexString := hex.EncodeToString(hash)
	signature := base64.StdEncoding.EncodeToString([]byte(hexString))
	return &Lmv1Token{
		AccessID:  accessID,
		Signature: signature,
		Epoch:     epochTime,
	}
}

func URL() (string, error) {
	company := os.Getenv("LM_ACCOUNT")
	if company == "" {
		if company = os.Getenv("LOGICMONITOR_ACCOUNT"); company == "" {
			return "", errors.New("environment variable `LM_ACCOUNT` or `LOGICMONITOR_ACCOUNT` must be provided")
		}
	}
	if company != "" {
		match, _ := regexp.MatchString(REGEX_COMPANY_NAME, company)
		if !match {
			return "", fmt.Errorf("invalid Company Name: %s", company)
		}
	}
	return fmt.Sprintf(ingestURL, company), nil
}

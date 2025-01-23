package model

import (
	"github.com/google/uuid"
)

type TraceIngestAPIResponse struct {
	StatusCode  int    `json:"statusCode"`
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	Error       error  `json:"error"`
	MultiStatus []struct {
		Code  float64 `json:"code"`
		Error string  `json:"error"`
	} `json:"multiStatus"`
	RetryAfter int `json:"retryAfter"`
}

type MetricsIngestAPIResponse struct {
	StatusCode  int    `json:"statusCode"`
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	Error       error  `json:"error"`
	MultiStatus []struct {
		Code  float64 `json:"code"`
		Error string  `json:"error"`
	} `json:"multiStatus"`
}

type LogsIngestAPIResponse struct {
	StatusCode  int    `json:"statusCode"`
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	Error       error  `json:"error"`
	MultiStatus []struct {
		Code  float64 `json:"code"`
		Error string  `json:"error"`
	} `json:"multiStatus"`
	RequestID  uuid.UUID `json:"requestId"`
	RetryAfter int       `json:"retryAfter"`
}

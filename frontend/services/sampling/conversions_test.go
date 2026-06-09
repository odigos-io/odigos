package sampling

import (
	"testing"

	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

func TestHeadSamplingOperationMatcherCRDToModelPreservesGrpc(t *testing.T) {
	matcher := &commonapisampling.HeadSamplingOperationMatcher{
		GrpcServer: &commonapisampling.HeadSamplingGrpcServerOperationMatcher{
			Method:  "Check",
			Service: "grpc.health.v1.Health",
		},
		GrpcClient: &commonapisampling.HeadSamplingGrpcClientOperationMatcher{
			Method:        "Export",
			Service:       "opentelemetry.proto.collector.trace.v1.TraceService",
			ServerAddress: "collector.odigos-system.svc",
		},
	}

	got := headSamplingOperationMatcherCRDToModel(matcher)
	if got == nil {
		t.Fatalf("expected model matcher")
	}
	if got.GrpcServer == nil {
		t.Fatalf("expected grpc server matcher to be preserved")
	}
	if got.GrpcServer.Method == nil || *got.GrpcServer.Method != "Check" {
		t.Fatalf("unexpected grpc server method: %#v", got.GrpcServer.Method)
	}
	if got.GrpcServer.Service == nil || *got.GrpcServer.Service != "grpc.health.v1.Health" {
		t.Fatalf("unexpected grpc server service: %#v", got.GrpcServer.Service)
	}
	if got.GrpcClient == nil {
		t.Fatalf("expected grpc client matcher to be preserved")
	}
	if got.GrpcClient.Method == nil || *got.GrpcClient.Method != "Export" {
		t.Fatalf("unexpected grpc client method: %#v", got.GrpcClient.Method)
	}
	if got.GrpcClient.Service == nil || *got.GrpcClient.Service != "opentelemetry.proto.collector.trace.v1.TraceService" {
		t.Fatalf("unexpected grpc client service: %#v", got.GrpcClient.Service)
	}
	if got.GrpcClient.ServerAddress == nil || *got.GrpcClient.ServerAddress != "collector.odigos-system.svc" {
		t.Fatalf("unexpected grpc client server address: %#v", got.GrpcClient.ServerAddress)
	}
}

func TestHeadSamplingOperationMatcherInputToCRDPreservesGrpc(t *testing.T) {
	method := "Check"
	service := "grpc.health.v1.Health"
	clientMethod := "Export"
	clientService := "opentelemetry.proto.collector.trace.v1.TraceService"
	serverAddress := "collector.odigos-system.svc"

	input := &model.HeadSamplingOperationMatcherInput{
		GrpcServer: &model.HeadSamplingGrpcServerMatcherInput{
			Method:  &method,
			Service: &service,
		},
		GrpcClient: &model.HeadSamplingGrpcClientMatcherInput{
			Method:        &clientMethod,
			Service:       &clientService,
			ServerAddress: &serverAddress,
		},
	}

	got := headSamplingOperationMatcherInputToCRD(input)
	if got == nil {
		t.Fatalf("expected CRD matcher")
	}
	if got.GrpcServer == nil {
		t.Fatalf("expected grpc server matcher to be preserved")
	}
	if got.GrpcServer.Method != method {
		t.Fatalf("unexpected grpc server method: %q", got.GrpcServer.Method)
	}
	if got.GrpcServer.Service != service {
		t.Fatalf("unexpected grpc server service: %q", got.GrpcServer.Service)
	}
	if got.GrpcClient == nil {
		t.Fatalf("expected grpc client matcher to be preserved")
	}
	if got.GrpcClient.Method != clientMethod {
		t.Fatalf("unexpected grpc client method: %q", got.GrpcClient.Method)
	}
	if got.GrpcClient.Service != clientService {
		t.Fatalf("unexpected grpc client service: %q", got.GrpcClient.Service)
	}
	if got.GrpcClient.ServerAddress != serverAddress {
		t.Fatalf("unexpected grpc client server address: %q", got.GrpcClient.ServerAddress)
	}
}

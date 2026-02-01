package csi

import (
	"context"
	"log/slog"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// HealthService implements grpc.health.v1.Health with actual CSI driver health checks
type HealthService struct {
	grpc_health_v1.UnimplementedHealthServer
	Identity *IdentityServer
}

// Check performs the health check by validating CSI driver readiness
func (h *HealthService) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	slog.Debug("Health check requested", "service", req.Service)

	// Check required paths using shared helper
	if !checkRequiredPaths() {
		return &grpc_health_v1.HealthCheckResponse{
			Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
		}, nil
	}

	// Also verify we can call our own CSI Identity service
	if _, err := h.Identity.GetPluginInfo(ctx, &csi.GetPluginInfoRequest{}); err != nil {
		slog.Debug("Health check failed - CSI Identity service not responding", "error", err)
		return &grpc_health_v1.HealthCheckResponse{
			Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
		}, nil
	}

	slog.Debug("Health check passed - CSI driver healthy")
	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}, nil
}

// Watch is required by the health interface but not used by grpc_health_probe
func (h *HealthService) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	return stream.Send(&grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	})
}

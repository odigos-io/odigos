package main

import (
	"context"
	"log/slog"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// IdentityServer implements the CSI Identity service interface.
//
// CSI Identity Service provides basic information about the driver:
// - GetPluginInfo: Returns driver name and version for kubelet discovery
// - GetPluginCapabilities: Declares what services this driver supports (Node only)
// - Probe: Health check to verify driver is ready to serve requests
//
// This is the first service kubelet calls to identify and validate the CSI driver.
// It's a prerequisite for kubelet to trust and use the driver for volume operations.
type IdentityServer struct {
	csi.UnimplementedIdentityServer
	name    string
	version string
}

func NewIdentityServer(name, version string) *IdentityServer {
	return &IdentityServer{
		name:    name,
		version: version,
	}
}

// GetPluginInfo returns the driver name and version.
// kubelet uses this to identify the driver and match it with CSI volume specs.
// The name "odigos.csi.driver" must match the driver field in pod CSI volumes.
func (s *IdentityServer) GetPluginInfo(ctx context.Context, req *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	slog.Debug("GetPluginInfo called")
	return &csi.GetPluginInfoResponse{
		Name:          s.name,
		VendorVersion: s.version,
	}, nil
}

// GetPluginCapabilities declares what services this driver implements.
// We only implement Node service (not Controller), so we return minimal capabilities.
// This tells kubelet we handle ephemeral inline volumes but not persistent volumes.
func (s *IdentityServer) GetPluginCapabilities(ctx context.Context, req *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	slog.Debug("GetPluginCapabilities called")

	return &csi.GetPluginCapabilitiesResponse{
		Capabilities: []*csi.PluginCapability{
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_UNKNOWN,
					},
				},
			},
		},
	}, nil
}

// Probe is a health check method called by kubelet to verify driver readiness.
// This method checks that all required host paths are accessible before declaring ready.
// If any critical path is missing, the driver reports not ready.
func (s *IdentityServer) Probe(ctx context.Context, req *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	slog.Debug("Probe called")

	// Use shared helper to check required paths
	if !checkRequiredPaths() {
		return &csi.ProbeResponse{
			Ready: &wrapperspb.BoolValue{Value: false},
		}, nil
	}

	slog.Debug("All required paths accessible, driver ready")
	return &csi.ProbeResponse{
		Ready: &wrapperspb.BoolValue{Value: true},
	}, nil
}

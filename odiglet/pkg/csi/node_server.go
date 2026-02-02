package csi

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NodeServer implements the CSI Node service interface.
//
// CSI Node Service is responsible for volume operations on individual nodes:
// - NodePublishVolume: Mount a volume to a specific path for a container
// - NodeUnpublishVolume: Unmount a volume from a path
// - NodeGetCapabilities: Declare what the node service can do
// - NodeGetInfo: Provide node-specific information
//
// In our case, we provide ephemeral inline volumes that mount instrumentation
// files from /var/odigos (host path) into containers at kubelet-specified paths.
// This replaces the device plugin approach with a standard CSI interface.
type NodeServer struct {
	csi.UnimplementedNodeServer
}

func NewNodeServer() *NodeServer {
	return &NodeServer{}
}

func (s *NodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (s *NodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// NodePublishVolume mounts the volume to the target path
//
// This is the core CSI method called by kubelet when a pod needs a volume mounted.
// The flow is:
//  1. Pod spec contains CSI volume with driver="odigos.csi.driver"
//  2. kubelet calls this method with targetPath (where to mount)
//  3. We bind mount /var/odigos to the targetPath
//  4. Container gets same instrumentation files as with device plugin
//
// This method reuses the existing deviceplugin mount logic as-is - same source path,
// same read-only semantics, just wrapped in CSI interface instead of device allocation.
func (s *NodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	volumeId := req.GetVolumeId()
	targetPath := req.GetTargetPath()

	// Basic validation
	if volumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	if targetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "Target path missing in request")
	}

	// Check if already mounted (idempotency)
	// We check in the HOST's mount namespace since that's where kubelet looks
	if isMounted, err := isPathMountedInHostNS(targetPath); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to check if path is mounted: %v", err))
	} else if isMounted {
		podUID := extractPodUIDFromPath(targetPath)
		slog.Info("Volume already mounted (idempotent)", "volumeId", volumeId, "podUID", podUID)
		return &csi.NodePublishVolumeResponse{}, nil
	}

	// The device plugin returns a Mount configuration that kubelet applies.
	// Here we directly apply that same mount operation.
	//
	// Original deviceplugin logic:
	//   ContainerPath: OdigosAgentsDir,  // "/var/odigos"
	//   HostPath:      OdigosAgentsDir,  // "/var/odigos"
	//   ReadOnly:      true,
	//
	// We perform the mount in the HOST's mount namespace using setns().
	// This allows kubelet to see the mount without requiring:
	//   - privileged: true
	//   - mountPropagation: Bidirectional
	// Instead, we only need CAP_SYS_ADMIN capability.

	sourcePath := k8sconsts.OdigosAgentsDirectory

	// Perform read-only bind mount in host namespace
	if err := mountBindInHostNS(sourcePath, targetPath); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to mount volume: %v", err))
	}

	podUID := extractPodUIDFromPath(targetPath)
	slog.Info("Successfully mounted volume in host namespace", "volumeId", req.GetVolumeId(), "podUID", podUID, "sourcePath", sourcePath)
	return &csi.NodePublishVolumeResponse{}, nil
}

// NodeUnpublishVolume is called during pod termination, AFTER containers stop.
// kubelet calls this to cleanup the mount and prevent stale mount points on the node.
func (s *NodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {

	// Basic validation
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	if len(req.GetTargetPath()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Target path missing in request")
	}

	targetPath := req.GetTargetPath()
	podUID := extractPodUIDFromPath(targetPath)

	// Check if mounted in host namespace (where kubelet looks)
	isMounted, err := isPathMountedInHostNS(targetPath)
	if err != nil {
		// Log but continue - we'll try to unmount anyway
		slog.Warn("Failed to check mount status, will try unmount anyway", "error", err, "podUID", podUID)
	}

	if !isMounted {
		// Even if not mounted, try to unmount anyway to handle stale/orphan mounts
		// that might not show up in /proc/mounts but still block pod termination
		if err := tryUnmountInHostNS(targetPath); err != nil {
			slog.Debug("Unmount attempt on non-mounted path", "targetPath", targetPath, "result", err)
		}
		slog.Info("Volume not mounted (idempotent)", "volumeId", req.GetVolumeId(), "podUID", podUID)
		return &csi.NodeUnpublishVolumeResponse{}, nil
	}

	// Unmount in host namespace
	if err := unmountInHostNS(targetPath); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to unmount: %v", err))
	}

	slog.Info("Successfully unmounted volume from host namespace", "volumeId", req.GetVolumeId(), "podUID", podUID)
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (s *NodeServer) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (s *NodeServer) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (s *NodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{
			{
				Type: &csi.NodeServiceCapability_Rpc{
					Rpc: &csi.NodeServiceCapability_RPC{
						Type: csi.NodeServiceCapability_RPC_UNKNOWN,
					},
				},
			},
		},
	}, nil
}

func (s *NodeServer) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	// Get node name from environment variable (set by Kubernetes)
	nodeID := os.Getenv(k8sconsts.NodeNameEnvVar)
	if nodeID == "" {
		return nil, status.Error(codes.Internal, k8sconsts.NodeNameEnvVar+" environment variable is required")
	}

	return &csi.NodeGetInfoResponse{
		NodeId: nodeID,
	}, nil
}

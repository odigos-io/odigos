package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/odigos-io/odigos/api/k8sconsts"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	odigletcsi "github.com/odigos-io/odigos/odiglet/pkg/csi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	v1 "k8s.io/kubelet/pkg/apis/pluginregistration/v1"
)

func main() {
	commonlogger.Init(os.Getenv("ODIGOS_LOG_LEVEL"), "odiglet")
	logger := commonlogger.LoggerCompat().With("subsystem", "csi-driver")
	logger.Info("Starting Odigos CSI Driver", "name", k8sconsts.OdigletCSIDriverName, "version", k8sconsts.OdigletCSIDriverVersion)

	// Create CSI driver
	driver := NewCSIDriver(k8sconsts.OdigletCSIDriverName, k8sconsts.OdigletCSIDriverVersion, logger)

	// Start gRPC server
	if err := driver.Run(logger); err != nil {
		logger.Error("Failed to start CSI driver", "err", err)
		os.Exit(1)
	}
}

type CSIDriver struct {
	name               string
	version            string
	server             *grpc.Server
	registrationServer *grpc.Server
	identity           *odigletcsi.IdentityServer
	node               *odigletcsi.NodeServer
}

func NewCSIDriver(name, version string, logger *commonlogger.OdigosLogger) *CSIDriver {
	return &CSIDriver{
		name:     name,
		version:  version,
		identity: odigletcsi.NewIdentityServer(name, version, logger),
		node:     odigletcsi.NewNodeServer(logger),
	}
}

func (d *CSIDriver) Run(logger *commonlogger.OdigosLogger) error {

	// Remove any existing socket file and ensure directory exists
	if err := os.Remove(k8sconsts.OdigletCSISocketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove socket file %s: %v", k8sconsts.OdigletCSISocketPath, err)
	}
	if err := os.MkdirAll(filepath.Dir(k8sconsts.OdigletCSISocketPath), 0750); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", filepath.Dir(k8sconsts.OdigletCSISocketPath), err)
	}

	lis, err := net.Listen("unix", k8sconsts.OdigletCSISocketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", k8sconsts.OdigletCSISocketPath, err)
	}

	d.server = grpc.NewServer()

	// Register CSI services
	csi.RegisterIdentityServer(d.server, d.identity) // Identity: driver info for kubelet discovery
	csi.RegisterNodeServer(d.server, d.node)         // Node: handles actual volume mount/unmount operations

	// Register custom health service that checks CSI driver health
	healthService := &odigletcsi.HealthService{Identity: d.identity, Logger: logger}
	grpc_health_v1.RegisterHealthServer(d.server, healthService)

	// Create context for coordinated shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start kubelet registration in background
	go func() {
		if err := d.registerWithKubelet(ctx, logger); err != nil && err != context.Canceled {
			logger.Error("Failed to register with kubelet", "err", err)
		}
	}()

	logger.Info("Listening on", "endpoint", k8sconsts.OdigletCSIEndpoint)

	// Handle shutdown gracefully
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		slog.Info("Received shutdown signal")

		// Stop both servers gracefully
		if d.registrationServer != nil {
			d.registrationServer.GracefulStop()
		}
		d.server.GracefulStop()
		cancel()
	}()

	return d.server.Serve(lis)
}

// registerWithKubelet registers the CSI driver with kubelet using the plugin registration API
func (d *CSIDriver) registerWithKubelet(ctx context.Context, logger *commonlogger.OdigosLogger) error {
	pluginRegistrationPath := k8sconsts.OdigletCSIRegistrationPath
	csiAddress := k8sconsts.OdigletCSISocketPath
	kubeletRegistrationPath := k8sconsts.KubeletPluginSocket

	// Create registration socket
	registrationPath := filepath.Join(pluginRegistrationPath, d.name+k8sconsts.OdigletCSIRegistrationSocketSuffix)
	if err := os.Remove(registrationPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove registration socket: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(registrationPath), 0750); err != nil {
		return fmt.Errorf("failed to create registration directory: %v", err)
	}

	lis, err := net.Listen("unix", registrationPath)
	if err != nil {
		return fmt.Errorf("failed to listen on registration socket: %v", err)
	}
	defer lis.Close()

	logger.Info("Starting kubelet registration", "socket", registrationPath)

	registrar := &nodeRegistrar{
		driverName:              d.name,
		endpoint:                csiAddress,
		kubeletRegistrationPath: kubeletRegistrationPath,
		logger:                  logger,
	}

	d.registrationServer = grpc.NewServer()
	v1.RegisterRegistrationServer(d.registrationServer, registrar)

	// Run server with context cancellation
	go func() {
		<-ctx.Done()
		d.registrationServer.GracefulStop()
	}()

	return d.registrationServer.Serve(lis)
}

// nodeRegistrar implements kubelet plugin registration
type nodeRegistrar struct {
	v1.UnimplementedRegistrationServer
	driverName              string
	endpoint                string
	kubeletRegistrationPath string
	logger                  *commonlogger.OdigosLogger
}

func (r *nodeRegistrar) GetInfo(ctx context.Context, req *v1.InfoRequest) (*v1.PluginInfo, error) {
	r.logger.Info("Registration GetInfo called")
	return &v1.PluginInfo{
		Type:              v1.CSIPlugin,
		Name:              r.driverName,
		Endpoint:          r.kubeletRegistrationPath,
		SupportedVersions: []string{"1.0.0"},
	}, nil
}

func (r *nodeRegistrar) NotifyRegistrationStatus(ctx context.Context, status *v1.RegistrationStatus) (*v1.RegistrationStatusResponse, error) {
	if !status.PluginRegistered {
		r.logger.Error("Registration failed", "message", status.Error)
		return nil, fmt.Errorf("registration failed: %s", status.Error)
	}

	r.logger.Info("CSI driver successfully registered with kubelet")
	return &v1.RegistrationStatusResponse{}, nil
}

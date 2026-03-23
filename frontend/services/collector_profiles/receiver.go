package collectorprofiles

import (
	"context"
	"log"
	"os"
	"strconv"
	"sync"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.opentelemetry.io/collector/receiver/xreceiver"
)

const (
	defaultProfilesPort = "4318"
	envProfilesEnabled  = "ENABLE_PROFILES_RECEIVER"
)

// Run starts the OTLP profiles receiver on a separate port (default 4318), the TTL cleanup goroutine,
// and blocks until ctx is done. If ENABLE_PROFILES_RECEIVER is false, it returns immediately.
func Run(ctx context.Context) {
	if !profilesReceiverEnabled() {
		log.Println("Profiles receiver disabled (set ENABLE_PROFILES_RECEIVER=true to enable)")
		return
	}

	maxSlots, ttlSec, slotMaxBytes, cleanupInt := StoreConfigFromEnv()
	store := NewProfileStore(maxSlots, ttlSec, slotMaxBytes, cleanupInt)
	store.RunCleanup(ctx)
	defer store.StopCleanup()

	profilesConsumer, err := NewProfilesConsumer(store)
	if err != nil {
		log.Printf("profiles: failed to create consumer: %v", err)
		return
	}

	f := otlpreceiver.NewFactory()
	cfg, ok := f.CreateDefaultConfig().(*otlpreceiver.Config)
	if !ok {
		log.Printf("profiles: failed to cast config to otlpreceiver.Config")
		return
	}
	// Profiles on a dedicated gRPC port so gateway can target it separately.
	cfg.GRPC = configoptional.Some(configgrpc.ServerConfig{
		NetAddr: confignet.AddrConfig{
			Endpoint:  "0.0.0.0:" + defaultProfilesPort,
			Transport: confignet.TransportTypeTCP,
		},
	})
	// Disable HTTP for this receiver to avoid port clash with default HTTP 4318.
	cfg.HTTP = configoptional.None[otlpreceiver.HTTPConfig]()

	xFactory, ok := f.(xreceiver.Factory)
	if !ok {
		log.Printf("profiles: otlpreceiver factory does not implement xreceiver.Factory")
		return
	}

	r, err := xFactory.CreateProfiles(ctx, receivertest.NewNopSettings(f.Type()), cfg, profilesConsumer)
	if err != nil {
		log.Printf("profiles: failed to create receiver: %v", err)
		return
	}

	if err := r.Start(ctx, componenttest.NewNopHost()); err != nil {
		log.Printf("profiles: failed to start receiver: %v", err)
		return
	}
	defer func() {
		if err := r.Shutdown(ctx); err != nil {
			log.Printf("profiles: shutdown error: %v", err)
		}
	}()

	log.Printf("OTLP profiles receiver is running on port %s", defaultProfilesPort)
	<-ctx.Done()
}

func profilesReceiverEnabled() bool {
	v := os.Getenv(envProfilesEnabled)
	if v == "" {
		return true
	}
	enabled, _ := strconv.ParseBool(v)
	return enabled
}

// ProfileStoreRef is a small interface for HTTP handlers that need StartViewing, GetProfileData, and optional DebugSlots.
type ProfileStoreRef interface {
	StartViewing(sourceKey string)
	GetProfileData(sourceKey string) [][]byte
	MaxSlots() int
	DebugSlots() (activeKeys []string, keysWithData []string)
}

// RunWithStore is like Run but accepts an existing store and returns it so the caller can pass it to HTTP handlers.
// The caller is responsible for starting the store's cleanup (store.RunCleanup(ctx)).
func RunWithStore(ctx context.Context, store *ProfileStore) (ProfileStoreRef, *sync.WaitGroup) {
	if !profilesReceiverEnabled() {
		log.Println("Profiles receiver disabled (set ENABLE_PROFILES_RECEIVER=true to enable)")
		return store, &sync.WaitGroup{}
	}

	profilesConsumer, err := NewProfilesConsumer(store)
	if err != nil {
		log.Printf("profiles: failed to create consumer: %v", err)
		return store, &sync.WaitGroup{}
	}

	f := otlpreceiver.NewFactory()
	cfg, ok := f.CreateDefaultConfig().(*otlpreceiver.Config)
	if !ok {
		log.Printf("profiles: failed to cast config to otlpreceiver.Config")
		return store, &sync.WaitGroup{}
	}
	cfg.GRPC = configoptional.Some(configgrpc.ServerConfig{
		NetAddr: confignet.AddrConfig{
			Endpoint:  "0.0.0.0:" + defaultProfilesPort,
			Transport: confignet.TransportTypeTCP,
		},
	})
	cfg.HTTP = configoptional.None[otlpreceiver.HTTPConfig]()

	xFactory, ok := f.(xreceiver.Factory)
	if !ok {
		log.Printf("profiles: otlpreceiver factory does not implement xreceiver.Factory")
		return store, &sync.WaitGroup{}
	}

	r, err := xFactory.CreateProfiles(ctx, receivertest.NewNopSettings(f.Type()), cfg, profilesConsumer)
	if err != nil {
		log.Printf("profiles: failed to create receiver: %v", err)
		return store, &sync.WaitGroup{}
	}

	if err := r.Start(ctx, componenttest.NewNopHost()); err != nil {
		log.Printf("profiles: failed to start receiver: %v", err)
		return store, &sync.WaitGroup{}
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		if err := r.Shutdown(ctx); err != nil {
			log.Printf("profiles: shutdown error: %v", err)
		}
	}()

	log.Printf("OTLP profiles receiver is running on port %s", defaultProfilesPort)
	return store, &wg
}

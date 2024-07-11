package connection

import (
	"sync"
	"time"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/opampserver/protobufs"
	"google.golang.org/protobuf/proto"
)

const (
	HeartbeatInterval = 30 * time.Second
)

// The time after which a connection is considered stale and can be cleaned up.
var connectionStaleTime = time.Duration(float64(HeartbeatInterval) * 2.5)

// Keep all live connections, with information about the connection.
// The cache is cleaned up periodically to expire opamp clients that are no longer connected reporting data.
type ConnectionsCache struct {
	mux sync.Mutex

	// map from device id to connection information
	liveConnections map[string]*ConnectionInfo
}

func NewConnectionsCache() *ConnectionsCache {
	return &ConnectionsCache{
		liveConnections: make(map[string]*ConnectionInfo),
	}
}

// GetConnection returns the connection information for the given device id.
// the returned object is a by-value copy of the connection information, so it can be safely used.
// To change something in the connection information, use the functions below which are synced and safe.
func (c *ConnectionsCache) GetConnection(deviceId string) (*ConnectionInfo, bool) {
	c.mux.Lock()
	defer c.mux.Unlock()
	conn, ok := c.liveConnections[deviceId]
	if !ok || conn == nil {
		return nil, false
	} else {
		// copy the conn object to avoid it being accessed concurrently
		connCopy := *conn
		return &connCopy, ok
	}
}

func (c *ConnectionsCache) AddConnection(deviceId string, conn *ConnectionInfo) {
	// copy the conn object to avoid it being accessed concurrently
	connCopy := *conn
	c.mux.Lock()
	defer c.mux.Unlock()
	c.liveConnections[deviceId] = &connCopy
}

func (c *ConnectionsCache) RemoveConnection(deviceId string) {
	c.mux.Lock()
	defer c.mux.Unlock()
	delete(c.liveConnections, deviceId)
}

func (c *ConnectionsCache) RecordMessageTime(deviceId string) {
	c.mux.Lock()
	defer c.mux.Unlock()

	conn, ok := c.liveConnections[deviceId]
	if !ok {
		return
	}
	conn.lastMessageTime = time.Now()
	c.liveConnections[deviceId] = conn
}

func (c *ConnectionsCache) CleanupStaleConnections() []ConnectionInfo {
	c.mux.Lock()
	defer c.mux.Unlock()

	deadConnectionInfos := make([]ConnectionInfo, 0)

	for deviceId, conn := range c.liveConnections {
		if time.Since(conn.lastMessageTime) > connectionStaleTime {
			delete(c.liveConnections, deviceId)
			deadConnectionInfos = append(deadConnectionInfos, *conn)
		}
	}

	return deadConnectionInfos
}

func (c *ConnectionsCache) UpdateRemoteConfig(workload common.PodWorkload, newConfigEntries *protobufs.AgentConfigMap) {
	c.mux.Lock()
	defer c.mux.Unlock()

	for _, conn := range c.liveConnections {
		if conn.Workload != workload {
			continue
		}

		// merge the new config entries into the existing remote config
		// copy the old remote config to avoid it being accessed concurrently
		newRemoteConfigMap := proto.Clone(conn.AgentRemoteConfig.Config).(*protobufs.AgentConfigMap)
		for key, value := range newConfigEntries.ConfigMap {
			newRemoteConfigMap.ConfigMap[key] = value
		}
		conn.AgentRemoteConfig = &protobufs.AgentRemoteConfig{
			Config:     newRemoteConfigMap,
			ConfigHash: CalcRemoteConfigHash(newRemoteConfigMap),
		}
	}
}

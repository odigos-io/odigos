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

// GetConnection returns the connection information for the given OpAMP instanceUid.
// the returned object is a by-value copy of the connection information, so it can be safely used.
// To change something in the connection information, use the functions below which are synced and safe.
func (c *ConnectionsCache) GetConnection(instanceUid string) (*ConnectionInfo, bool) {
	c.mux.Lock()
	defer c.mux.Unlock()
	conn, ok := c.liveConnections[instanceUid]
	if !ok || conn == nil {
		return nil, false
	} else {
		// copy the conn object to avoid it being accessed concurrently
		connCopy := *conn
		return &connCopy, ok
	}
}

func (c *ConnectionsCache) AddConnection(instanceUid string, conn *ConnectionInfo) {
	// copy the conn object to avoid it being accessed concurrently
	connCopy := *conn
	c.mux.Lock()
	defer c.mux.Unlock()
	c.liveConnections[instanceUid] = &connCopy
}

func (c *ConnectionsCache) RemoveConnection(instanceUid string) {
	c.mux.Lock()
	defer c.mux.Unlock()
	delete(c.liveConnections, instanceUid)
}

func (c *ConnectionsCache) RecordMessageTime(instanceUid string) {
	c.mux.Lock()
	defer c.mux.Unlock()

	conn, ok := c.liveConnections[instanceUid]
	if !ok {
		return
	}
	conn.lastMessageTime = time.Now()
	c.liveConnections[instanceUid] = conn
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

// allow to completely overwrite the remote config for a set of keys for a given workload
func (c *ConnectionsCache) UpdateWorkloadRemoteConfigByKeys(workload common.PodWorkload, newConfigEntries *protobufs.AgentConfigMap) {
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

// how to use this function:
// it allows you to update remote config keys which will be sent to the agent on next heartbeat.
// should be used to update general odigos pipeline configs that are shared by all connections.
// for example: updating the enabled signals configuration.
// pass it a callback, that will be called for each connection, and should return the new config entries for that connection.
// entries that are not specified in the returned map will retain their previous value.
// each key should be updated entirely when specified, partial updates are not supported.
// the callback should be fast and not block, as it will be called for each connection with the lock held.
func (c *ConnectionsCache) UpdateAllConnectionConfigs(connConfigEvaluator func(connInfo *ConnectionInfo) *protobufs.AgentConfigMap) {
	c.mux.Lock()
	defer c.mux.Unlock()

	for _, conn := range c.liveConnections {
		newConfigEntries := connConfigEvaluator(conn)
		if newConfigEntries == nil {
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

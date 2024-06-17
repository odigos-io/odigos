package server

import (
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
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

type ConnectionInfo struct {
	DeviceId        string
	Pod             *corev1.Pod
	lastMessageTime time.Time
}

func NewConnectionsCache() *ConnectionsCache {
	return &ConnectionsCache{
		liveConnections: make(map[string]*ConnectionInfo),
	}
}

// GetConnection returns the connection information for the given device id.
func (c *ConnectionsCache) GetConnection(deviceId string) (*ConnectionInfo, bool) {
	c.mux.Lock()
	defer c.mux.Unlock()
	conn, ok := c.liveConnections[deviceId]
	return conn, ok
}

func (c *ConnectionsCache) AddConnection(deviceId string, conn *ConnectionInfo) {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.liveConnections[deviceId] = conn
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

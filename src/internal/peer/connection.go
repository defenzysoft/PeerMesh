package peer

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// Connection represents a connection to a peer
type Connection struct {
	ID            string
	RemoteAddr    net.Addr
	LocalAddr     net.Addr
	Conn          net.Conn
	IsConnected   bool
	LastSeen      time.Time
	BytesSent     int64
	BytesReceived int64
	Latency       time.Duration
	mutex         sync.RWMutex
}

// ConnectionPool manages all peer connections
type ConnectionPool struct {
	connections map[string]*Connection
	mutex       sync.RWMutex
	config      PoolConfig
}

// PoolConfig holds configuration for the connection pool
type PoolConfig struct {
	MaxConnections int
	Timeout        time.Duration
	KeepAlive      time.Duration
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(config PoolConfig) *ConnectionPool {
	if config.MaxConnections <= 0 {
		config.MaxConnections = 100
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.KeepAlive == 0 {
		config.KeepAlive = 60 * time.Second
	}

	return &ConnectionPool{
		connections: make(map[string]*Connection),
		config:      config,
	}
}

// Connect establishes a connection to a peer
func (p *ConnectionPool) Connect(peerID string, addr string) (*Connection, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if conn, exists := p.connections[peerID]; exists && conn.IsConnected {
		return conn, nil
	}

	if len(p.connections) >= p.config.MaxConnections {
		return nil, fmt.Errorf("connection pool is full")
	}

	conn, err := net.DialTimeout("tcp", addr, p.config.Timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	connection := &Connection{
		ID:          peerID,
		RemoteAddr:  conn.RemoteAddr(),
		LocalAddr:   conn.LocalAddr(),
		Conn:        conn,
		IsConnected: true,
		LastSeen:    time.Now(),
	}

	p.connections[peerID] = connection

	log.Printf("Connected to peer %s at %s", peerID, addr)
	return connection, nil
}

// Disconnect disconnects from a peer
func (p *ConnectionPool) Disconnect(peerID string) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	conn, exists := p.connections[peerID]
	if !exists {
		return fmt.Errorf("connection to peer %s not found", peerID)
	}

	if conn.Conn != nil {
		err := conn.Conn.Close()
		if err != nil {
			log.Printf("Error closing connection to %s: %v", peerID, err)
		}
	}

	conn.IsConnected = false
	delete(p.connections, peerID)

	log.Printf("Disconnected from peer %s", peerID)
	return nil
}

// GetConnection returns a connection to a specific peer
func (p *ConnectionPool) GetConnection(peerID string) *Connection {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.connections[peerID]
}

// GetAllConnections returns all active connections
func (p *ConnectionPool) GetAllConnections() []*Connection {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	var connections []*Connection
	for _, conn := range p.connections {
		if conn.IsConnected {
			connections = append(connections, conn)
		}
	}
	return connections
}

// Send sends data to a peer
func (p *ConnectionPool) Send(peerID string, data []byte) error {
	conn := p.GetConnection(peerID)
	if conn == nil {
		return fmt.Errorf("no connection to peer %s", peerID)
	}

	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	if !conn.IsConnected {
		return fmt.Errorf("connection to peer %s is not active", peerID)
	}

	err := conn.Conn.SetWriteDeadline(time.Now().Add(p.config.Timeout))
	if err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}

	n, err := conn.Conn.Write(data)
	if err != nil {
		conn.IsConnected = false
		return fmt.Errorf("failed to send data: %w", err)
	}

	conn.BytesSent += int64(n)
	conn.LastSeen = time.Now()

	return nil
}

// Receive receives data from a peer
func (p *ConnectionPool) Receive(peerID string, buffer []byte) (int, error) {
	conn := p.GetConnection(peerID)
	if conn == nil {
		return 0, fmt.Errorf("no connection to peer %s", peerID)
	}

	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	if !conn.IsConnected {
		return 0, fmt.Errorf("connection to peer %s is not active", peerID)
	}

	err := conn.Conn.SetReadDeadline(time.Now().Add(p.config.Timeout))
	if err != nil {
		return 0, fmt.Errorf("failed to set read deadline: %w", err)
	}

	n, err := conn.Conn.Read(buffer)
	if err != nil {
		conn.IsConnected = false
		return 0, fmt.Errorf("failed to receive data: %w", err)
	}

	conn.BytesReceived += int64(n)
	conn.LastSeen = time.Now()

	return n, nil
}

// Ping sends a ping to a peer and measures latency
func (p *ConnectionPool) Ping(peerID string) (time.Duration, error) {
	conn := p.GetConnection(peerID)
	if conn == nil {
		return 0, fmt.Errorf("no connection to peer %s", peerID)
	}

	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	if !conn.IsConnected {
		return 0, fmt.Errorf("connection to peer %s is not active", peerID)
	}

	start := time.Now()
	pingData := []byte("ping")

	err := conn.Conn.SetWriteDeadline(time.Now().Add(p.config.Timeout))
	if err != nil {
		return 0, fmt.Errorf("failed to set write deadline: %w", err)
	}

	_, err = conn.Conn.Write(pingData)
	if err != nil {
		conn.IsConnected = false
		return 0, fmt.Errorf("failed to send ping: %w", err)
	}

	err = conn.Conn.SetReadDeadline(time.Now().Add(p.config.Timeout))
	if err != nil {
		return 0, fmt.Errorf("failed to set read deadline: %w", err)
	}

	buffer := make([]byte, 4)
	_, err = conn.Conn.Read(buffer)
	if err != nil {
		conn.IsConnected = false
		return 0, fmt.Errorf("failed to receive pong: %w", err)
	}

	latency := time.Since(start)
	conn.Latency = latency
	conn.LastSeen = time.Now()

	return latency, nil
}

// IsConnected checks if a connection to a peer is active
func (p *ConnectionPool) IsConnected(peerID string) bool {
	conn := p.GetConnection(peerID)
	if conn == nil {
		return false
	}

	conn.mutex.RLock()
	defer conn.mutex.RUnlock()
	return conn.IsConnected
}

// GetStats returns connection pool statistics
func (p *ConnectionPool) GetStats() map[string]interface{} {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	totalConnections := len(p.connections)
	activeConnections := 0
	totalBytesSent := int64(0)
	totalBytesReceived := int64(0)

	for _, conn := range p.connections {
		if conn.IsConnected {
			activeConnections++
		}
		totalBytesSent += conn.BytesSent
		totalBytesReceived += conn.BytesReceived
	}

	return map[string]interface{}{
		"total_connections":    totalConnections,
		"active_connections":   activeConnections,
		"max_connections":      p.config.MaxConnections,
		"total_bytes_sent":     totalBytesSent,
		"total_bytes_received": totalBytesReceived,
		"timeout":              p.config.Timeout,
		"keep_alive":           p.config.KeepAlive,
	}
}

// Cleanup removes inactive connections
func (p *ConnectionPool) Cleanup() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	now := time.Now()
	var toRemove []string

	for peerID, conn := range p.connections {
		if now.Sub(conn.LastSeen) > p.config.KeepAlive {
			toRemove = append(toRemove, peerID)
		}
	}

	for _, peerID := range toRemove {
		conn := p.connections[peerID]
		if conn.Conn != nil {
			conn.Conn.Close()
		}
		delete(p.connections, peerID)
		log.Printf("Removed inactive connection to peer %s", peerID)
	}
}

// Close closes all connections
func (p *ConnectionPool) Close() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for peerID, conn := range p.connections {
		if conn.Conn != nil {
			err := conn.Conn.Close()
			if err != nil {
				log.Printf("Error closing connection to %s: %v", peerID, err)
			}
		}
		conn.IsConnected = false
	}

	p.connections = make(map[string]*Connection)
	log.Printf("Closed all peer connections")
	return nil
}

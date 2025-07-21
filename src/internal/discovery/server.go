package discovery

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Server represents a discovery server that manages peer connections
type Server struct {
	addr            string
	peers           map[string]*PeerConnection
	mutex           sync.RWMutex
	upgrader        websocket.Upgrader
	stopChan        chan struct{}
	isRunning       bool
	heartbeatTicker *time.Ticker
	cleanupTicker   *time.Ticker
}

// PeerConnection represents a connected peer
type PeerConnection struct {
	ID       string
	Conn     *websocket.Conn
	Peer     *Peer
	LastSeen time.Time
	IsOnline bool
}

// ServerConfig holds configuration for the discovery server
type ServerConfig struct {
	Addr              string
	HeartbeatInterval time.Duration
	CleanupInterval   time.Duration
}

// NewServer creates a new discovery server
func NewServer(config ServerConfig) (*Server, error) {
	if config.Addr == "" {
		config.Addr = ":8080"
	}
	if config.HeartbeatInterval == 0 {
		config.HeartbeatInterval = 30 * time.Second
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 60 * time.Second
	}

	server := &Server{
		addr:     config.Addr,
		peers:    make(map[string]*PeerConnection),
		stopChan: make(chan struct{}),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		heartbeatTicker: time.NewTicker(config.HeartbeatInterval),
		cleanupTicker:   time.NewTicker(config.CleanupInterval),
	}

	return server, nil
}

// Start starts the discovery server
func (s *Server) Start() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.isRunning {
		return fmt.Errorf("server is already running")
	}

	// Start background goroutines
	go s.heartbeatLoop()
	go s.cleanupLoop()

	// Setup HTTP handler
	http.HandleFunc("/ws", s.handleWebSocket)

	// Start HTTP server in background
	go func() {
		log.Printf("Discovery server starting on %s", s.addr)
		err := http.ListenAndServe(s.addr, nil)
		if err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Give server a moment to start
	time.Sleep(10 * time.Millisecond)

	s.isRunning = true
	return nil
}

// Stop stops the discovery server
func (s *Server) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.isRunning {
		return nil
	}

	s.isRunning = false
	close(s.stopChan)

	s.heartbeatTicker.Stop()
	s.cleanupTicker.Stop()

	s.mutex.Lock()
	for _, peerConn := range s.peers {
		peerConn.Conn.Close()
	}
	s.peers = make(map[string]*PeerConnection)
	s.mutex.Unlock()

	log.Println("Discovery server stopped")
	return nil
}

// IsRunning returns whether the server is running
func (s *Server) IsRunning() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.isRunning
}

// GetPeers returns a list of all connected peers
func (s *Server) GetPeers() []*Peer {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var peers []*Peer
	for _, peerConn := range s.peers {
		if peerConn.IsOnline {
			peers = append(peers, peerConn.Peer)
		}
	}
	return peers
}

// GetPeer returns a specific peer by ID
func (s *Server) GetPeer(peerID string) *Peer {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if peerConn, exists := s.peers[peerID]; exists {
		return peerConn.Peer
	}
	return nil
}

// handleWebSocket handles WebSocket connections
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	log.Printf("New WebSocket connection from %s", r.RemoteAddr)

	go s.handleConnection(conn)
}

// handleConnection handles messages from a specific peer connection
func (s *Server) handleConnection(conn *websocket.Conn) {
	defer func() {
		conn.Close()
		log.Printf("WebSocket connection closed")
	}()

	for {
		select {
		case <-s.stopChan:
			return
		default:
			_, data, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Read message error: %v", err)
				s.removePeer(conn)
				return
			}

			var msg Message
			err = json.Unmarshal(data, &msg)
			if err != nil {
				log.Printf("Failed to unmarshal message: %v", err)
				continue
			}

			err = s.handleMessage(conn, &msg)
			if err != nil {
				log.Printf("Failed to handle message: %v", err)
			}
		}
	}
}

// handleMessage handles incoming messages from peers
func (s *Server) handleMessage(conn *websocket.Conn, msg *Message) error {
	switch msg.Type {
	case "register":
		return s.handleRegister(conn, msg)
	case "ping":
		return s.handlePing(conn, msg)
	case "pong":
		return s.handlePong(conn, msg)
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

// handleRegister handles peer registration
func (s *Server) handleRegister(conn *websocket.Conn, msg *Message) error {
	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid register payload")
	}

	peerID, ok := payload["peer_id"].(string)
	if !ok {
		return fmt.Errorf("missing peer_id in register payload")
	}

	localIPStr, ok := payload["local_ip"].(string)
	if !ok {
		return fmt.Errorf("missing local_ip in register payload")
	}

	remoteAddr := conn.RemoteAddr().String()
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}

	peerIP := net.ParseIP(host)
	if peerIP == nil {
		peerIP = net.IPv4(0, 0, 0, 0)
	}

	peer := &Peer{
		ID:       peerID,
		IP:       peerIP,
		Port:     0,
		LastSeen: time.Now(),
		IsOnline: true,
		LocalIP:  localIPStr,
	}

	peerConn := &PeerConnection{
		ID:       peerID,
		Conn:     conn,
		Peer:     peer,
		LastSeen: time.Now(),
		IsOnline: true,
	}

	s.mutex.Lock()
	s.peers[peerID] = peerConn
	s.mutex.Unlock()

	log.Printf("Peer registered: %s (%s)", peerID, localIPStr)

	err = s.sendPeerList(conn)
	if err != nil {
		return fmt.Errorf("failed to send peer list: %w", err)
	}

	err = s.broadcastPeerJoined(peer)
	if err != nil {
		return fmt.Errorf("failed to broadcast peer joined: %w", err)
	}

	return nil
}

// handlePing handles ping messages
func (s *Server) handlePing(conn *websocket.Conn, msg *Message) error {
	s.updatePeerLastSeen(conn)

	pong := &Message{
		Type: "pong",
	}

	data, err := json.Marshal(pong)
	if err != nil {
		return fmt.Errorf("failed to marshal pong: %w", err)
	}

	return conn.WriteMessage(websocket.TextMessage, data)
}

// handlePong handles pong messages
func (s *Server) handlePong(conn *websocket.Conn, msg *Message) error {
	s.updatePeerLastSeen(conn)
	return nil
}

// sendPeerList sends the current peer list to a specific connection
func (s *Server) sendPeerList(conn *websocket.Conn) error {
	s.mutex.RLock()
	var peers []*Peer
	for _, peerConn := range s.peers {
		if peerConn.IsOnline {
			peers = append(peers, peerConn.Peer)
		}
	}
	s.mutex.RUnlock()

	msg := &Message{
		Type: "peer_list",
		Payload: map[string]interface{}{
			"peers": peers,
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal peer list: %w", err)
	}

	return conn.WriteMessage(websocket.TextMessage, data)
}

// broadcastPeerJoined notifies all peers about a new peer
func (s *Server) broadcastPeerJoined(peer *Peer) error {
	msg := &Message{
		Type: "peer_joined",
		Payload: map[string]interface{}{
			"id":        peer.ID,
			"ip":        peer.IP,
			"port":      peer.Port,
			"public_ip": peer.PublicIP,
			"local_ip":  peer.LocalIP,
		},
	}

	return s.broadcastMessage(msg)
}

// broadcastPeerLeft notifies all peers about a peer leaving
func (s *Server) broadcastPeerLeft(peerID string) error {
	msg := &Message{
		Type: "peer_left",
		Payload: map[string]interface{}{
			"id": peerID,
		},
	}

	return s.broadcastMessage(msg)
}

// broadcastMessage sends a message to all connected peers
func (s *Server) broadcastMessage(msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, peerConn := range s.peers {
		if peerConn.IsOnline {
			err := peerConn.Conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				log.Printf("Failed to send message to peer %s: %v", peerConn.ID, err)
			}
		}
	}

	return nil
}

// removePeer removes a peer from the registry
func (s *Server) removePeer(conn *websocket.Conn) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var peerID string
	for id, peerConn := range s.peers {
		if peerConn.Conn == conn {
			peerID = id
			break
		}
	}

	if peerID != "" {
		delete(s.peers, peerID)
		log.Printf("Peer removed: %s", peerID)

		go s.broadcastPeerLeft(peerID)
	}
}

// updatePeerLastSeen updates the last seen time for a peer
func (s *Server) updatePeerLastSeen(conn *websocket.Conn) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, peerConn := range s.peers {
		if peerConn.Conn == conn {
			peerConn.LastSeen = time.Now()
			peerConn.Peer.LastSeen = time.Now()
			break
		}
	}
}

// heartbeatLoop periodically checks peer health
func (s *Server) heartbeatLoop() {
	for {
		select {
		case <-s.stopChan:
			return
		case <-s.heartbeatTicker.C:
			s.sendHeartbeats()
		}
	}
}

// sendHeartbeats sends ping messages to all peers
func (s *Server) sendHeartbeats() {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	ping := &Message{
		Type: "ping",
	}

	data, err := json.Marshal(ping)
	if err != nil {
		log.Printf("Failed to marshal ping: %v", err)
		return
	}

	for _, peerConn := range s.peers {
		if peerConn.IsOnline {
			err := peerConn.Conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				log.Printf("Failed to send ping to peer %s: %v", peerConn.ID, err)
				peerConn.IsOnline = false
			}
		}
	}
}

// cleanupLoop periodically removes dead peers
func (s *Server) cleanupLoop() {
	for {
		select {
		case <-s.stopChan:
			return
		case <-s.cleanupTicker.C:
			s.cleanupDeadPeers()
		}
	}
}

// cleanupDeadPeers removes peers that haven't responded recently
func (s *Server) cleanupDeadPeers() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	timeout := 2 * time.Minute

	var deadPeers []string
	for peerID, peerConn := range s.peers {
		if now.Sub(peerConn.LastSeen) > timeout {
			deadPeers = append(deadPeers, peerID)
		}
	}

	for _, peerID := range deadPeers {
		peerConn := s.peers[peerID]
		peerConn.Conn.Close()
		delete(s.peers, peerID)
		log.Printf("Removed dead peer: %s", peerID)

		go s.broadcastPeerLeft(peerID)
	}
}

// GetStats returns server statistics
func (s *Server) GetStats() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	onlinePeers := 0
	for _, peerConn := range s.peers {
		if peerConn.IsOnline {
			onlinePeers++
		}
	}

	return map[string]interface{}{
		"running":       s.isRunning,
		"addr":          s.addr,
		"total_peers":   len(s.peers),
		"online_peers":  onlinePeers,
		"offline_peers": len(s.peers) - onlinePeers,
	}
}

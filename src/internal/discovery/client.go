package discovery

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net"
	"net/url"
	"sync"
	"time"
)

type Client struct {
	serverURL   string
	conn        *websocket.Conn
	peerID      string
	localIP     net.IP
	peers       map[string]*Peer
	mutex       sync.RWMutex
	stopChan    chan struct{}
	isConnected bool
	reconnect   bool
	handlers    map[string]MessageHandler
}

type Peer struct {
	ID       string    `json:"id"`
	IP       net.IP    `json:"ip"`
	Port     int       `json:"port"`
	LastSeen time.Time `json:"last_seen"`
	IsOnline bool      `json:"is_online"`
	PublicIP string    `json:"public_ip,omitempty"`
	LocalIP  string    `json:"local_ip,omitempty"`
}

type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
	From    string      `json:"from,omitempty"`
}

type MessageHandler func(*Message) error

type ClientConfig struct {
	ServerURL string
	PeerID    string
	LocalIP   net.IP
	Reconnect bool
}

// NewClient creates a new discovery client
func NewClient(config ClientConfig) (*Client, error) {
	if config.ServerURL == "" {
		return nil, fmt.Errorf("server URL is required")
	}

	if config.PeerID == "" {
		return nil, fmt.Errorf("peer ID is required")
	}

	client := &Client{
		serverURL: config.ServerURL,
		peerID:    config.PeerID,
		localIP:   config.LocalIP,
		peers:     make(map[string]*Peer),
		stopChan:  make(chan struct{}),
		reconnect: config.Reconnect,
		handlers:  make(map[string]MessageHandler),
	}

	client.registerDefaultHandlers()

	return client, nil
}

func (c *Client) Connect() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.isConnected {
		return fmt.Errorf("already connected")
	}

	u, err := url.Parse(c.serverURL)
	if err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect to discovery server: %w", err)
	}

	c.conn = conn
	c.isConnected = true

	go c.handleMessages()

	err = c.sendRegistration()
	if err != nil {
		err := c.conn.Close()
		if err != nil {
			return fmt.Errorf("could not close connection: %w", err)
		}
		c.isConnected = false
		return fmt.Errorf("failed to register with server: %w", err)
	}

	log.Printf("Connected to discovery server %s", c.serverURL)
	return nil
}

func (c *Client) Disconnect() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.isConnected {
		return nil
	}

	c.isConnected = false
	close(c.stopChan)

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}

	return nil
}

func (c *Client) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.isConnected
}

// GetPeers return a list of all discovered peers
func (c *Client) GetPeers() []*Peer {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var peers []*Peer
	for _, peer := range c.peers {
		peers = append(peers, peer)
	}
	return peers
}

// GetPeer returns a specific peer by ID
func (c *Client) GetPeer(peerID string) *Peer {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.peers[peerID]
}

// RegisterMessageHandler registers a handler for a specific message type
func (c *Client) RegisterMessageHandler(messageType string, handler MessageHandler) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.handlers[messageType] = handler
}

// SendMessage sends a message to the discovery server
func (c *Client) SendMessage(msg *Message) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected to discovery server")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	err = c.conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

func (c *Client) handleMessages() {
	for {
		select {
		case <-c.stopChan:
			return
		default:
			_, data, err := c.conn.ReadMessage()
			if err != nil {
				if c.reconnect {
					log.Printf("Connection lost, attempting to reconnect...")
					c.reconnectToServer()
				} else {
					log.Printf("Connection error: %v", err)
				}
				continue
			}

			var msg Message
			err = json.Unmarshal(data, &msg)
			if err != nil {
				log.Printf("Failed to unmarshal message: %v", err)
				continue
			}

			err = c.handleMessage(&msg)
			if err != nil {
				log.Printf("Failed to handle message: %v", err)
			}
		}
	}
}

func (c *Client) handleMessage(msg *Message) error {
	c.mutex.RLock()
	handler, exists := c.handlers[msg.Type]
	c.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("no handler for message type: %s", msg.Type)
	}

	return handler(msg)
}

// sendRegistration sends a registration message to the server
func (c *Client) sendRegistration() error {
	registration := &Message{
		Type: "register",
		Payload: map[string]interface{}{
			"peer_id":  c.peerID,
			"local_ip": c.localIP.String(),
		},
	}

	return c.SendMessage(registration)
}

func (c *Client) reconnectToServer() {
	for {
		select {
		case <-c.stopChan:
			return
		default:
			time.Sleep(5 * time.Second)

			err := c.Connect()
			if err == nil {
				log.Printf("Successfully connected to discovery server")
				return
			}

			log.Printf("Reconnect failed: %v", err)
		}
	}
}

func (c *Client) registerDefaultHandlers() {
	c.RegisterMessageHandler("peer_list", c.handlePeerList)
	c.RegisterMessageHandler("peer_joined", c.handlePeerJoined)
	c.RegisterMessageHandler("peer_left", c.handlePeerLeft)
	c.RegisterMessageHandler("ping", c.handlePing)
}

func (c *Client) handlePeerList(msg *Message) error {
	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid peer list payload")
	}

	peersData, ok := payload["peers"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid peers data")
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.peers = make(map[string]*Peer)

	for _, peerData := range peersData {
		peerMap, ok := peerData.(map[string]interface{})
		if !ok {
			continue
		}

		peer := &Peer{
			ID:       peerMap["id"].(string),
			IP:       net.ParseIP(peerMap["ip"].(string)),
			Port:     int(peerMap["port"].(float64)),
			LastSeen: time.Now(),
			IsOnline: true,
		}

		if publicIP, ok := peerMap["public_ip"].(string); ok {
			peer.PublicIP = publicIP
		}

		if localIP, ok := peerMap["local_ip"].(string); ok {
			peer.LocalIP = localIP
		}

		c.peers[peer.ID] = peer
	}

	log.Printf("Updated peer list: %d peers", len(c.peers))
	return nil
}

func (c *Client) handlePeerJoined(msg *Message) error {
	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid peer joined payload")
	}

	peer := &Peer{
		ID:       payload["id"].(string),
		IP:       net.ParseIP(payload["ip"].(string)),
		Port:     int(payload["port"].(float64)),
		LastSeen: time.Now(),
		IsOnline: true,
	}

	if publicIP, ok := payload["public_ip"].(string); ok {
		peer.PublicIP = publicIP
	}

	if localIP, ok := payload["local_ip"].(string); ok {
		peer.LocalIP = localIP
	}

	c.mutex.Lock()
	c.peers[peer.ID] = peer
	c.mutex.Unlock()

	log.Printf("Peer joined %s (%s)", peer.ID, peer.IP.String())
	return nil
}

func (c *Client) handlePeerLeft(msg *Message) error {
	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid peer left payload")
	}

	peerID := payload["id"].(string)

	c.mutex.Lock()
	if peer, exists := c.peers[peerID]; exists {
		peer.IsOnline = false
		peer.LastSeen = time.Now()
		log.Printf("Peer left %s (%s)", peer.ID, peer.IP.String())
	}
	c.mutex.Unlock()
	return nil
}

// handlePing handles ping messages
func (c *Client) handlePing(msg *Message) error {
	pong := &Message{
		Type: "pong",
		From: c.peerID,
	}

	return c.SendMessage(pong)
}

// GetStats returns client statistics
func (c *Client) GetStats() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	onlinePeers := 0
	for _, peer := range c.peers {
		if peer.IsOnline {
			onlinePeers++
		}
	}

	return map[string]interface{}{
		"connected":     c.isConnected,
		"peer_id":       c.peerID,
		"server_url":    c.serverURL,
		"total_peers":   len(c.peers),
		"online_peers":  onlinePeers,
		"offline_peers": len(c.peers) - onlinePeers,
	}
}

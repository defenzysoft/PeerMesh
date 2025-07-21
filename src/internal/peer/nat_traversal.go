package peer

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// NATTraversal handles NAT traversal using UDP hole punching
type NATTraversal struct {
	localAddr   *net.UDPAddr
	conn        *net.UDPConn
	stunServers []string
	mutex       sync.RWMutex
	peers       map[string]*PeerInfo
}

// PeerInfo holds information about a peer for NAT traversal
type PeerInfo struct {
	ID          string
	PublicAddr  *net.UDPAddr
	LocalAddr   *net.UDPAddr
	LastSeen    time.Time
	IsReachable bool
}

// TraversalConfig holds configuration for NAT traversal
type TraversalConfig struct {
	LocalPort        int
	STUNServers      []string
	HolePunchTimeout time.Duration
}

// NewNATTraversal creates a new NAT traversal instance
func NewNATTraversal(config TraversalConfig) (*NATTraversal, error) {
	if config.LocalPort == 0 {
		config.LocalPort = 0
	}
	if len(config.STUNServers) == 0 {
		config.STUNServers = []string{
			"stun.l.google.com:19302",
			"stun1.l.google.com:19302",
			"stun.stunprotocol.org:3478",
		}
	}
	if config.HolePunchTimeout == 0 {
		config.HolePunchTimeout = 30 * time.Second
	}

	localAddr := &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: config.LocalPort,
	}

	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP listener: %w", err)
	}

	traversal := &NATTraversal{
		localAddr:   conn.LocalAddr().(*net.UDPAddr),
		conn:        conn,
		stunServers: config.STUNServers,
		peers:       make(map[string]*PeerInfo),
	}

	log.Printf("NAT traversal initialized on %s", traversal.localAddr.String())
	return traversal, nil
}

// GetPublicAddress discovers the public IP address using STUN
func (t *NATTraversal) GetPublicAddress() (*net.UDPAddr, error) {
	for _, stunServer := range t.stunServers {
		addr, err := t.querySTUNServer(stunServer)
		if err == nil {
			log.Printf("Public address discovered: %s", addr.String())
			return addr, nil
		}
		log.Printf("STUN query to %s failed: %v", stunServer, err)
	}

	return nil, fmt.Errorf("failed to discover public address from any STUN server")
}

// querySTUNServer queries a STUN server for public address
func (t *NATTraversal) querySTUNServer(server string) (*net.UDPAddr, error) {
	request := t.createSTUNRequest()

	serverAddr, err := net.ResolveUDPAddr("udp", server)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve STUN server: %w", err)
	}

	_, err = t.conn.WriteToUDP(request, serverAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to send STUN request: %w", err)
	}

	t.conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	buffer := make([]byte, 1024)
	n, _, err := t.conn.ReadFromUDP(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to receive STUN response: %w", err)
	}

	return t.parseSTUNResponse(buffer[:n])
}

// createSTUNRequest creates a STUN binding request
func (t *NATTraversal) createSTUNRequest() []byte {
	request := make([]byte, 20)

	binary.BigEndian.PutUint16(request[0:2], 0x0001)
	binary.BigEndian.PutUint16(request[2:4], 0x0000)
	binary.BigEndian.PutUint32(request[4:8], 0x2112A442)

	for i := 8; i < 20; i++ {
		request[i] = 0
	}

	return request
}

// parseSTUNResponse parses a STUN response to extract public address
func (t *NATTraversal) parseSTUNResponse(data []byte) (*net.UDPAddr, error) {
	if len(data) < 20 {
		return nil, fmt.Errorf("STUN response too short")
	}

	messageType := binary.BigEndian.Uint16(data[0:2])
	if messageType != 0x0101 {
		return nil, fmt.Errorf("unexpected STUN message type: 0x%04x", messageType)
	}

	offset := 20
	for offset < len(data)-4 {
		attrType := binary.BigEndian.Uint16(data[offset : offset+2])
		attrLen := binary.BigEndian.Uint16(data[offset+2 : offset+4])

		if attrType == 0x0020 {
			if offset+4+int(attrLen) > len(data) {
				return nil, fmt.Errorf("STUN attribute too short")
			}

			attrData := data[offset+4 : offset+4+int(attrLen)]
			return t.parseXORMappedAddress(attrData)
		}

		offset += 4 + int(attrLen)
	}

	return nil, fmt.Errorf("XOR-MAPPED-ADDRESS not found in STUN response")
}

// parseXORMappedAddress parses XOR-MAPPED-ADDRESS attribute
func (t *NATTraversal) parseXORMappedAddress(data []byte) (*net.UDPAddr, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("XOR-MAPPED-ADDRESS too short")
	}

	family := data[1]
	if family != 0x01 {
		return nil, fmt.Errorf("unsupported address family: %d", family)
	}

	port := binary.BigEndian.Uint16(data[2:4]) ^ 0x2112

	ipBytes := make([]byte, 4)
	for i := 0; i < 4; i++ {
		ipBytes[i] = data[4+i] ^ 0x21
	}

	ip := net.IP(ipBytes)

	return &net.UDPAddr{
		IP:   ip,
		Port: int(port),
	}, nil
}

// StartHolePunch initiates UDP hole punching with a peer
func (t *NATTraversal) StartHolePunch(peerID string, peerPublicAddr *net.UDPAddr) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.peers[peerID] = &PeerInfo{
		ID:          peerID,
		PublicAddr:  peerPublicAddr,
		LastSeen:    time.Now(),
		IsReachable: false,
	}

	go t.performHolePunch(peerID, peerPublicAddr)

	return nil
}

// performHolePunch performs the actual UDP hole punching
func (t *NATTraversal) performHolePunch(peerID string, peerAddr *net.UDPAddr) {
	log.Printf("Starting UDP hole punch to %s (%s)", peerID, peerAddr.String())

	for i := 0; i < 5; i++ {
		punchData := []byte(fmt.Sprintf("PUNCH:%s", t.localAddr.String()))
		_, err := t.conn.WriteToUDP(punchData, peerAddr)
		if err != nil {
			log.Printf("Failed to send punch packet %d: %v", i+1, err)
		}

		time.Sleep(100 * time.Millisecond)
	}

	t.listenForPunchResponse(peerID, peerAddr)
}

// listenForPunchResponse listens for punch response from peer
func (t *NATTraversal) listenForPunchResponse(peerID string, peerAddr *net.UDPAddr) {
	timeout := time.After(10 * time.Second)

	for {
		select {
		case <-timeout:
			log.Printf("Hole punch timeout for peer %s", peerID)
			return
		default:
			t.conn.SetReadDeadline(time.Now().Add(1 * time.Second))

			buffer := make([]byte, 1024)
			n, remoteAddr, err := t.conn.ReadFromUDP(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				log.Printf("Error reading punch response: %v", err)
				continue
			}

			if remoteAddr.String() == peerAddr.String() {
				data := string(buffer[:n])
				if len(data) > 6 && data[:6] == "PUNCH:" {
					log.Printf("Hole punch successful with peer %s", peerID)

					t.mutex.Lock()
					if peer, exists := t.peers[peerID]; exists {
						peer.IsReachable = true
						peer.LastSeen = time.Now()
					}
					t.mutex.Unlock()

					return
				}
			}
		}
	}
}

// IsPeerReachable checks if a peer is reachable
func (t *NATTraversal) IsPeerReachable(peerID string) bool {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	if peer, exists := t.peers[peerID]; exists {
		return peer.IsReachable
	}
	return false
}

// GetPeerInfo returns information about a peer
func (t *NATTraversal) GetPeerInfo(peerID string) *PeerInfo {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return t.peers[peerID]
}

// GetAllPeers returns all tracked peers
func (t *NATTraversal) GetAllPeers() []*PeerInfo {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	var peers []*PeerInfo
	for _, peer := range t.peers {
		peers = append(peers, peer)
	}
	return peers
}

// Close closes the NAT traversal instance
func (t *NATTraversal) Close() error {
	if t.conn != nil {
		return t.conn.Close()
	}
	return nil
}

// GetStats returns NAT traversal statistics
func (t *NATTraversal) GetStats() map[string]interface{} {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	reachablePeers := 0
	for _, peer := range t.peers {
		if peer.IsReachable {
			reachablePeers++
		}
	}

	return map[string]interface{}{
		"local_address":   t.localAddr.String(),
		"total_peers":     len(t.peers),
		"reachable_peers": reachablePeers,
		"stun_servers":    t.stunServers,
	}
}

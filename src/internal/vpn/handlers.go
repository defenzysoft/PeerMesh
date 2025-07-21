package vpn

import (
	"log"
)

// ICMPHandler handles ICMP packets
type ICMPHandler struct {
	manager *Manager
}

// HandlePacket handles an ICMP packet
func (h *ICMPHandler) HandlePacket(packet *Packet) error {
	log.Printf("ICMP packet: %s -> %s", packet.Source.String(), packet.Dest.String())

	// TODO: Implement proper ICMP handling (ping, etc.)
	return nil
}

// TCPHandler handles TCP packets
type TCPHandler struct {
	manager *Manager
}

// HandlePacket handles a TCP packet
func (h *TCPHandler) HandlePacket(packet *Packet) error {
	log.Printf("TCP packet: %s -> %s", packet.Source.String(), packet.Dest.String())

	// TODO: Implement TCP connection tracking and forwarding
	return nil
}

// UDPHandler handles UDP packets
type UDPHandler struct {
	manager *Manager
}

// HandlePacket handles a UDP packet
func (h *UDPHandler) HandlePacket(packet *Packet) error {
	log.Printf("UDP packet: %s -> %s", packet.Source.String(), packet.Dest.String())

	// TODO: Implement UDP connection tracking and forwarding
	return nil
}

// DefaultHandler handles unknown packet types
type DefaultHandler struct {
	manager *Manager
}

// HandlePacket handles unknown packet types
func (h *DefaultHandler) HandlePacket(packet *Packet) error {
	log.Printf("Unknown packet type: %s -> %s", packet.Source.String(), packet.Dest.String())
	return nil
}

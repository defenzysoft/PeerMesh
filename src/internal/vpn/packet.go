package vpn

import (
	"encoding/binary"
	"fmt"
	"net"
)

// PacketType represents the type of packet
type PacketType int

const (
	PacketTypeUnknown PacketType = iota
	PacketTypeIPv4
	PacketTypeIPv6
	PacketTypeICMP
	PacketTypeTCP
	PacketTypeUDP
)

// Packet represents a network packet
type Packet struct {
	Data   []byte
	Length int
	Source net.IP
	Dest   net.IP
	Type   PacketType
}

// PacketHandler defines the interface for packet processing
type PacketHandler interface {
	HandlePacket(packet *Packet) error
}

// PacketProcessor handles packet parsing and routing
type PacketProcessor struct {
	handlers map[PacketType]PacketHandler
	buffer   []byte
}

// NewPacketProcessor creates a new packet processor
func NewPacketProcessor(bufferSize int) *PacketProcessor {
	if bufferSize <= 0 {
		bufferSize = 2048
	}

	return &PacketProcessor{
		handlers: make(map[PacketType]PacketHandler),
		buffer:   make([]byte, bufferSize),
	}
}

// RegisterHandler registers a packet handler for a specific packet type
func (p *PacketProcessor) RegisterHandler(packetType PacketType, handler PacketHandler) {
	p.handlers[packetType] = handler
}

// ProcessPacket processes a raw packet
func (p *PacketProcessor) ProcessPacket(data []byte) error {
	packet, err := p.parsePacket(data)
	if err != nil {
		return fmt.Errorf("failed to parse packet: %w", err)
	}

	handler, exists := p.handlers[packet.Type]
	if !exists {
		return p.handleUnknownPacket(packet)
	}

	return handler.HandlePacket(packet)
}

// parsePacket parses a raw packet and extracts basic information
func (p *PacketProcessor) parsePacket(data []byte) (*Packet, error) {
	if len(data) < 20 {
		return nil, fmt.Errorf("packet too short")
	}

	packet := &Packet{
		Data:   data,
		Length: len(data),
	}

	version := (data[0] >> 4) & 0x0F
	switch version {
	case 4:
		return p.parseIPv4Packet(packet)
	case 6:
		return p.parseIPv6Packet(packet)
	default:
		return nil, fmt.Errorf("unsupported IP version: %d", version)
	}
}

// parseIPv4Packet parses an IPv4 packet
func (p *PacketProcessor) parseIPv4Packet(packet *Packet) (*Packet, error) {
	if len(packet.Data) < 20 {
		return nil, fmt.Errorf("IPv4 packet too short")
	}

	packet.Source = net.IP(packet.Data[12:16])
	packet.Dest = net.IP(packet.Data[16:20])

	protocol := packet.Data[9]
	switch protocol {
	case 1: // ICMP
		packet.Type = PacketTypeICMP
	case 6: // TCP
		packet.Type = PacketTypeTCP
	case 17: // UDP
		packet.Type = PacketTypeUDP
	default:
		packet.Type = PacketTypeIPv4
	}

	return packet, nil
}

// parseIPv6Packet parses an IPv6 packet
func (p *PacketProcessor) parseIPv6Packet(packet *Packet) (*Packet, error) {
	if len(packet.Data) < 40 {
		return nil, fmt.Errorf("IPv6 packet too short")
	}

	packet.Source = net.IP(packet.Data[8:24])
	packet.Dest = net.IP(packet.Data[24:40])

	nextHeader := packet.Data[6]
	switch nextHeader {
	case 1: // ICMPv6
		packet.Type = PacketTypeICMP
	case 6: // TCP
		packet.Type = PacketTypeTCP
	case 17: // UDP
		packet.Type = PacketTypeUDP
	default:
		packet.Type = PacketTypeIPv6
	}

	return packet, nil
}

// handleUnknownPacket handles packets with unknown types
func (p *PacketProcessor) handleUnknownPacket(packet *Packet) error {
	// TODO: Implement proper handling or forwarding
	return nil
}

// GetPacketSource returns the source IP of a packet
func (p *Packet) GetPacketSource() net.IP {
	return p.Source
}

// GetPacketDest returns the destination IP of a packet
func (p *Packet) GetPacketDest() net.IP {
	return p.Dest
}

// GetPacketType returns the packet type
func (p *Packet) GetPacketType() PacketType {
	return p.Type
}

// IsLocalTraffic checks if the packet is for local traffic
func (p *Packet) IsLocalTraffic(localIP net.IP) bool {
	return p.Dest.Equal(localIP)
}

// IsBroadcast checks if the packet is a broadcast packet
func (p *Packet) IsBroadcast() bool {
	if len(p.Dest) == 4 {
		return p.Dest.Equal(net.IPv4bcast)
	}
	return false
}

// IsMulticast checks if the packet is a multicast packet
func (p *Packet) IsMulticast() bool {
	if len(p.Dest) == 4 {
		return p.Dest[0] >= 224 && p.Dest[0] <= 239
	} else if len(p.Dest) == 16 {
		return p.Dest[0] == 0xff
	}
	return false
}

// GetTTL returns the TTL value for IPv4 packets
func (p *Packet) GetTTL() (int, error) {
	if len(p.Data) < 9 {
		return 0, fmt.Errorf("packet too short to get TTL")
	}
	return int(p.Data[8]), nil
}

// SetTTL sets the TTL value for IPv4 packets
func (p *Packet) SetTTL(ttl int) error {
	if len(p.Data) < 9 {
		return fmt.Errorf("packet too short to set TTL")
	}
	if ttl < 0 || ttl > 255 {
		return fmt.Errorf("invalid TTL value: %d", ttl)
	}
	p.Data[8] = byte(ttl)
	return nil
}

// CalculateChecksum calculates the IP checksum
func (p *Packet) CalculateChecksum() (uint16, error) {
	if len(p.Data) < 20 {
		return 0, fmt.Errorf("packet too short to calculate checksum")
	}

	p.Data[10] = 0
	p.Data[11] = 0

	var sum uint32
	for i := 0; i < 20; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(p.Data[i : i+2]))
	}

	for sum > 0xffff {
		sum = (sum & 0xffff) + (sum >> 16)
	}

	return uint16(^sum), nil
}

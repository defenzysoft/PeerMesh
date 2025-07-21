package vpn

import (
	"fmt"
	"net"
	"sync"
)

// IPAllocator manages IP address allocation within a subnet
type IPAllocator struct {
	subnet    *net.IPNet
	allocated map[string]bool
	mutex     sync.RWMutex
	nextIP    net.IP
}

// NewIPAllocator creates a new IP allocator for the given subnet
func NewIPAllocator(subnet *net.IPNet) (*IPAllocator, error) {
	if subnet == nil {
		return nil, fmt.Errorf("subnet cannot be nil")
	}

	// Calculate the first usable IP (network address + 1)
	firstIP := make(net.IP, len(subnet.IP))
	copy(firstIP, subnet.IP)

	// Increment the first IP to get the first usable address
	for i := len(firstIP) - 1; i >= 0; i-- {
		firstIP[i]++
		if firstIP[i] != 0 {
			break
		}
	}

	allocator := &IPAllocator{
		subnet:    subnet,
		allocated: make(map[string]bool),
		nextIP:    firstIP,
	}

	return allocator, nil
}

// AllocateIP allocates the next available IP address
func (a *IPAllocator) AllocateIP() (net.IP, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	allocatedIP := a.findNextAvailableIP()
	if allocatedIP == nil {
		return nil, fmt.Errorf("no available IP addresses in subnet %s", a.subnet.String())
	}

	ipStr := allocatedIP.String()
	a.allocated[ipStr] = true

	return allocatedIP, nil
}

// AllocateSpecificIP allocates a specific IP address if available
func (a *IPAllocator) AllocateSpecificIP(ip net.IP) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if !a.subnet.Contains(ip) {
		return fmt.Errorf("IP %s is not in subnet %s", ip.String(), a.subnet.String())
	}

	if a.isNetworkOrBroadcast(ip) {
		return fmt.Errorf("IP %s is network or broadcast address", ip.String())
	}

	ipStr := ip.String()
	if a.allocated[ipStr] {
		return fmt.Errorf("IP %s is already allocated", ip.String())
	}

	a.allocated[ipStr] = true
	return nil
}

// ReleaseIP releases an allocated IP address
func (a *IPAllocator) ReleaseIP(ip net.IP) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	ipStr := ip.String()
	if !a.allocated[ipStr] {
		return fmt.Errorf("IP %s is not allocated", ip.String())
	}

	delete(a.allocated, ipStr)
	return nil
}

// IsAllocated checks if an IP address is allocated
func (a *IPAllocator) IsAllocated(ip net.IP) bool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	ipStr := ip.String()
	return a.allocated[ipStr]
}

// GetSubnet returns the subnet managed by this allocator
func (a *IPAllocator) GetSubnet() *net.IPNet {
	return a.subnet
}

// GetAvailableCount returns the number of available IP addresses
func (a *IPAllocator) GetAvailableCount() int {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	total := a.getTotalIPs()
	allocated := len(a.allocated)

	reserved := 2
	return total - allocated - reserved
}

// GetAllocatedIPs returns a list of all allocated IP addresses
func (a *IPAllocator) GetAllocatedIPs() []net.IP {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	var ips []net.IP
	for ipStr := range a.allocated {
		ip := net.ParseIP(ipStr)
		if ip != nil {
			ips = append(ips, ip)
		}
	}
	return ips
}

// findNextAvailableIP finds the next available IP address
func (a *IPAllocator) findNextAvailableIP() net.IP {
	currentIP := make(net.IP, len(a.nextIP))
	copy(currentIP, a.nextIP)

	totalIPs := a.getTotalIPs()
	for i := 0; i < totalIPs; i++ {
		if !a.isNetworkOrBroadcast(currentIP) {
			ipStr := currentIP.String()
			if !a.allocated[ipStr] {
				a.nextIP = a.incrementIP(currentIP)
				return currentIP
			}
		}
		currentIP = a.incrementIP(currentIP)
	}

	return nil
}

// incrementIP increments an IP address by 1
func (a *IPAllocator) incrementIP(ip net.IP) net.IP {
	nextIP := make(net.IP, len(ip))
	copy(nextIP, ip)

	for i := len(nextIP) - 1; i >= 0; i-- {
		nextIP[i]++
		if nextIP[i] != 0 {
			break
		}
	}

	return nextIP
}

// isNetworkOrBroadcast checks if an IP is the network or broadcast address
func (a *IPAllocator) isNetworkOrBroadcast(ip net.IP) bool {
	if ip.Equal(a.subnet.IP) {
		return true
	}

	broadcast := a.getBroadcastAddress()
	if ip.Equal(broadcast) {
		return true
	}

	return false
}

// getBroadcastAddress calculates the broadcast address for the subnet
func (a *IPAllocator) getBroadcastAddress() net.IP {
	broadcast := make(net.IP, len(a.subnet.IP))
	copy(broadcast, a.subnet.IP)

	mask := a.subnet.Mask
	for i := range broadcast {
		broadcast[i] |= ^mask[i]
	}

	return broadcast
}

// getTotalIPs calculates the total number of IP addresses in the subnet
func (a *IPAllocator) getTotalIPs() int {
	ones, bits := a.subnet.Mask.Size()
	return 1 << (bits - ones)
}

// GetGatewayIP returns a suggested gateway IP (first usable IP)
func (a *IPAllocator) GetGatewayIP() net.IP {
	gateway := make(net.IP, len(a.subnet.IP))
	copy(gateway, a.subnet.IP)

	for i := len(gateway) - 1; i >= 0; i-- {
		gateway[i]++
		if gateway[i] != 0 {
			break
		}
	}

	return gateway
}

// GetDNSIPs returns suggested DNS IPs (second and third usable IPs)
func (a *IPAllocator) GetDNSIPs() []net.IP {
	gateway := a.GetGatewayIP()

	dns1 := a.incrementIP(gateway)
	dns2 := a.incrementIP(dns1)

	return []net.IP{dns1, dns2}
}

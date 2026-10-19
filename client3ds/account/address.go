package account

import (
	"fmt"
	"strconv"
	"strings"
)

// DefaultPort is the server's port when the address leaves it out, as the server's own default.
const DefaultPort = 3000

// secureSchemes ask for a secure connection when the server is typed with one.
var secureSchemes = []string{"https://", "wss://"}

// Address is a server as the player typed it, understood.
type Address struct {
	Host   string
	Port   int
	Secure bool // talk TLS: it was typed with https:// (or wss://)
}

// HostPort is the address as a Host header writes it.
func (address Address) HostPort() string { return address.Host + ":" + strconv.Itoa(address.Port) }

// ParseAddress understands a server typed as name:port, address:port or https://name:port. A
// server at home is typed without https:// and talked to in plain text; the port is DefaultPort
// when left out.
func ParseAddress(typed string) (Address, error) {
	typed = strings.TrimSpace(typed)
	address := Address{Port: DefaultPort}
	for _, scheme := range secureSchemes {
		if len(typed) >= len(scheme) && strings.EqualFold(typed[:len(scheme)], scheme) {
			typed, address.Secure = typed[len(scheme):], true
		}
	}
	typed = strings.TrimSuffix(typed, "/")
	host, portText, found := strings.Cut(typed, ":")
	if host == "" {
		return Address{}, fmt.Errorf("%w: no server name in %q", ErrAddress, typed)
	}
	address.Host = host
	if !found {
		return address, nil
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port <= 0 || port > 65535 {
		return Address{}, fmt.Errorf("%w: the port %q is not a number from 1 to 65535", ErrAddress, portText)
	}
	address.Port = port
	return address, nil
}

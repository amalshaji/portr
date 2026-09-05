package main

import (
	"errors"
	"net"
	"strconv"
)

// parseLocalTarget reads the positional target of the one-off http and tcp
// commands. It accepts a bare port ("8000") or a host:port pair
// ("192.168.1.50:8000", "[::1]:8000"), with the port required to be in the
// 1-65535 range. An empty host means the tunnel's default, which
// Tunnel.SetDefaults resolves to localhost.
func parseLocalTarget(arg string) (string, int, error) {
	host, portStr, err := net.SplitHostPort(arg)
	switch {
	case err != nil:
		// No host part: treat the whole argument as a bare port.
		host, portStr = "", arg
	case host == "":
		return "", 0, errInvalidTarget
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return "", 0, errInvalidTarget
	}
	return host, port, nil
}

var errInvalidTarget = errors.New("please specify a port (1-65535) or a host:port target (use [addr]:port for IPv6)")

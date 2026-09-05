package main

import (
	"errors"
	"net"
	"strconv"
)

// parseLocalTarget reads the positional target of the one-off http and tcp
// commands. It accepts a bare port ("8000") or a host:port pair
// ("192.168.1.50:8000", "[::1]:8000"). An empty host means the tunnel's
// default, which Tunnel.SetDefaults resolves to localhost.
func parseLocalTarget(arg string) (host string, port int, err error) {
	if p, atoiErr := strconv.Atoi(arg); atoiErr == nil {
		return "", p, nil
	}

	host, portStr, err := net.SplitHostPort(arg)
	if err != nil || host == "" {
		return "", 0, errInvalidTarget
	}
	port, err = strconv.Atoi(portStr)
	if err != nil {
		return "", 0, errInvalidTarget
	}
	return host, port, nil
}

var errInvalidTarget = errors.New("please specify a port or a host:port target (use [addr]:port for IPv6)")

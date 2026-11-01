package utils

import (
	"net"
	"strings"
)

// IsLoopbackHost reports whether host, with or without a port, names this
// machine: localhost or a loopback IP.
func IsLoopbackHost(host string) bool {
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// +build !android

package utils

import (
	"net"
	"time"
)

var VpnMode bool

func Dial(network, address string) (net.Conn, error) {
	return net.Dial(network, address)
}

func DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout(network, address, timeout)
}

func DialTCP(network string, laddr, raddr *net.TCPAddr) (*net.TCPConn, error) {
	return net.DialTCP(network, laddr, raddr)
}

func DialUDP(network string, laddr, raddr *net.UDPAddr) (*net.UDPConn, error) {
	return net.DialUDP(network, laddr, raddr)
}

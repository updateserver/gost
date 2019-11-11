// +build android

package utils

/*
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/time.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <sys/un.h>
#include <sys/uio.h>

#define ANCIL_FD_BUFFER(n) \
    struct { \
        struct cmsghdr h; \
        int fd[n]; \
    }

int
ancil_send_fds_with_buffer(int sock, const int *fds, unsigned n_fds, void *buffer)
{
    struct msghdr msghdr;
    char nothing = '!';
    struct iovec nothing_ptr;
    struct cmsghdr *cmsg;
    int i;

    nothing_ptr.iov_base = &nothing;
    nothing_ptr.iov_len = 1;
    msghdr.msg_name = NULL;
    msghdr.msg_namelen = 0;
    msghdr.msg_iov = &nothing_ptr;
    msghdr.msg_iovlen = 1;
    msghdr.msg_flags = 0;
    msghdr.msg_control = buffer;
    msghdr.msg_controllen = sizeof(struct cmsghdr) + sizeof(int) * n_fds;
    cmsg = CMSG_FIRSTHDR(&msghdr);
    cmsg->cmsg_len = msghdr.msg_controllen;
    cmsg->cmsg_level = SOL_SOCKET;
    cmsg->cmsg_type = SCM_RIGHTS;
    for(i = 0; i < n_fds; i++)
        ((int *)CMSG_DATA(cmsg))[i] = fds[i];
    return(sendmsg(sock, &msghdr, 0) >= 0 ? 0 : -1);
}

int
ancil_send_fd(int sock, int fd)
{
    ANCIL_FD_BUFFER(1) buffer;

    return(ancil_send_fds_with_buffer(sock, &fd, 1, &buffer));
}

void
set_timeout(int sock)
{
    struct timeval tv;
    tv.tv_sec  = 3;
    tv.tv_usec = 0;
    setsockopt(sock, SOL_SOCKET, SO_RCVTIMEO, (char *)&tv, sizeof(struct timeval));
    setsockopt(sock, SOL_SOCKET, SO_SNDTIMEO, (char *)&tv, sizeof(struct timeval));
}

*/
import "C"

import (
	"context"
	"log"
	"net"
	"syscall"
	"time"
)

var VpnMode bool

func ControlOnConnSetup(network string, address string, c syscall.RawConn) error {
	if VpnMode {
		fn := func(s uintptr) {
			fd := int(s)
			path := "protect_path"

			socket, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
			if err != nil {
				log.Println(err)
				return
			}

			defer syscall.Close(socket)

			C.set_timeout(C.int(socket))

			err = syscall.Connect(socket, &syscall.SockaddrUnix{Name: path})
			if err != nil {
				log.Println(err)
				return
			}

			C.ancil_send_fd(C.int(socket), C.int(fd))

			dummy := []byte{1}
			n, err := syscall.Read(socket, dummy)
			if err != nil {
				log.Println(err)
				return
			}
			if n != 1 {
				log.Println("Failed to protect fd: ", fd)
				return
			}
		}

		if err := c.Control(fn); err != nil {
			return err
		}
	}

	return nil
}

func Dial(network, address string) (net.Conn, error) {
	if !VpnMode {
		return net.Dial(network, address)
	}
	log.Printf("Dial %v %v setup", network, address)
	d := net.Dialer{Control: ControlOnConnSetup}
	return d.Dial(network, address)
}

func DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if !VpnMode {
		d := net.Dialer{}
		return d.DialContext(ctx, network, address)
	}
	log.Printf("DialContext %v %v setup", network, address)
	d := net.Dialer{Control: ControlOnConnSetup}
	return d.DialContext(ctx, network, address)
}

func init() {
	net.DefaultResolver = &net.Resolver{Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		log.Printf("DefaultResolver address %v modify to %v", address, "114.114.114.114:53")
		return DialContext(ctx, network, "119.29.29.29:53")
	}, PreferGo: true}
}

func DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	if !VpnMode {
		return net.DialTimeout(network, address, timeout)
	}
	log.Printf("DialTimeout %v setup", address)
	d := net.Dialer{Control: ControlOnConnSetup, Timeout: timeout}
	return d.Dial(network, address)
}

func DialTCP(network string, laddr, raddr *net.TCPAddr) (*net.TCPConn, error) {
	if !VpnMode {
		return net.DialTCP(network, laddr, raddr)
	}
	if laddr != nil {
		log.Println("DialUDP laddr not nil fixme.")
	}
	log.Printf("DialTCP %v setup", raddr.String())
	d := net.Dialer{Control: ControlOnConnSetup}
	conn, err := d.Dial(network, raddr.String())
	return conn.(*net.TCPConn), err
}

func DialUDP(network string, laddr, raddr *net.UDPAddr) (*net.UDPConn, error) {
	if !VpnMode {
		return net.DialUDP(network, laddr, raddr)
	}
	if laddr != nil {
		log.Println("DialUDP laddr not nil fixme.")
	}
	log.Printf("DialUDP %v setup", raddr.String())
	d := net.Dialer{Control: ControlOnConnSetup}
	conn, err := d.Dial(network, raddr.String())
	return conn.(*net.UDPConn), err
}

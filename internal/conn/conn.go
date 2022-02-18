package conn

import (
	"io"
	"log"
	"net"
)

const (
	KeepLive      = "KEEP_LIVE"
	NewConnection = "NEW_CONNECTION"
)

func ConnectServer(addr string) (*net.TCPConn, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	tcpConn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return nil, err
	}
	return tcpConn, nil
}

// Pipe2Conn src ->dst,dst->src
func Pipe2Conn(src net.Conn, dst net.Conn) {
	go copyConn(dst, src)
	copyConn(src, dst)
}

func copyConn(src net.Conn, dst net.Conn) {
	defer src.Close()
	defer dst.Close()

	_, err := io.Copy(dst, src)
	if err != nil {
		log.Printf("failed to copy data,err:%s", err.Error())
		return
	}
}

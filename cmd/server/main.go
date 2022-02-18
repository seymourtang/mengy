package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"

	"github.com/seymourtang/mengy/internal/conn"
	"github.com/seymourtang/mengy/internal/listener"
)

var (
	ProxyPort uint

	ControllerPort uint
	TransferPort   uint
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.UintVar(&ProxyPort, "proxyPort", 8081, "the port of proxy service")
	flag.UintVar(&ControllerPort, "controllerPort", 8082, "the port of server controller")
	flag.UintVar(&TransferPort, "transferPort", 8083, "the port of server transfer")
	flag.Parse()

	const localIP = "0.0.0.0"
	var (
		controllerAddr = fmt.Sprintf("%s:%d", localIP, ControllerPort)
		transferAddr   = fmt.Sprintf("%s:%d", localIP, TransferPort)
		proxyAddr      = fmt.Sprintf("%s:%d", localIP, ProxyPort)
	)
	quit := make(chan os.Signal, 1)
	stop := make(chan struct{})
	signal.Notify(quit, os.Interrupt)
	go func() {
		<-quit
		stop <- struct{}{}
	}()
	s := NewServer(controllerAddr, transferAddr, proxyAddr)
	s.Run(stop)
}

type Server struct {
	clientConn     net.Conn
	controllerAddr string
	transferAddr   string
	proxyAddr      string
	proxyLocker    sync.Mutex
	proxyConnStore map[string]net.Conn
}

func NewServer(controllerAddr string, transferAddr string, proxyAddr string) *Server {
	return &Server{
		controllerAddr: controllerAddr,
		transferAddr:   transferAddr,
		proxyAddr:      proxyAddr,
		proxyConnStore: map[string]net.Conn{},
	}
}

func (s *Server) Run(stop <-chan struct{}) {
	go s.StartControllerServer()
	go s.StartProxyServer()
	go s.StartTransferServer()
	<-stop
	log.Println("exiting...")
}

func (s *Server) StartControllerServer() {
	server, err := listener.StartTCPServer(s.controllerAddr)
	if err != nil {
		log.Fatalf("failed to start controller server,err:%s", err.Error())
		return
	}
	log.Printf("listen to controller server in %s", s.controllerAddr)
	defer server.Close()
	for {
		conn, err := server.Accept()
		if err != nil {
			log.Printf("accept a new conn,%s", err.Error())
			continue
		}
		s.clientConn = conn
	}
}

func (s *Server) StartProxyServer() {
	server, err := listener.StartTCPServer(s.proxyAddr)
	if err != nil {
		log.Fatalf("failed to start proxy server,err:%s", err.Error())
		return
	}
	log.Printf("listen to proxy server in %s", s.proxyAddr)
	for {
		conn, err := server.Accept()
		if err != nil {
			log.Printf("accept a new conn,%s", err.Error())
			continue
		}
		s.proxyLocker.Lock()
		s.proxyConnStore[conn.RemoteAddr().String()] = conn
		s.proxyLocker.Unlock()
		s.SendNotifyMsg(conn.RemoteAddr().String())
	}

}

func (s *Server) SendNotifyMsg(key string) {
	if s.clientConn == nil {
		return
	}
	_, err := s.clientConn.Write([]byte(conn.NewConnection + "\n"))
	if err != nil {
		log.Printf("failed to send NewConnection msg,err:%s", err.Error())
	}
	_, err = s.clientConn.Write([]byte(key + "\n"))
	if err != nil {
		log.Printf("failed to send key(%s) msg,err:%s", key, err.Error())
	}

}

func (s *Server) StartTransferServer() {
	server, err := listener.StartTCPServer(s.transferAddr)
	if err != nil {
		log.Fatalf("failed to start transfer server,err:%s", err.Error())
		return
	}
	log.Printf("listen to transfer server in %s", s.transferAddr)
	defer server.Close()
	for {
		c, err := server.Accept()
		if err != nil {
			log.Printf("accept a new conn,%s", err.Error())
			continue
		}
		reader := bufio.NewReader(c)
		requestAddr, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("accept a new conn,%s", err.Error())

		}
		log.Printf("received key:%s", requestAddr)
		reqAddr := strings.TrimRight(requestAddr, "\n")
		s.proxyLocker.Lock()
		proxyConn := s.proxyConnStore[reqAddr]
		s.proxyLocker.Unlock()
		go conn.Pipe2Conn(proxyConn, c)
		s.proxyLocker.Lock()
		delete(s.proxyConnStore, reqAddr)
		s.proxyLocker.Unlock()
	}

}

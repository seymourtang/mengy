package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"

	"github.com/seymourtang/mengy/internal/conn"
)

var (
	localServiceIP   string
	localServicePort uint

	RemoteIP       string
	ControllerPort uint
	TransferPort   uint
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.StringVar(&localServiceIP, "localServiceIP", "localhost", "the IP address of local service")
	flag.UintVar(&localServicePort, "localServicePort", 9091, "the port of local service")
	flag.StringVar(&RemoteIP, "remoteIP", "localhost", "the IP address of remote server")
	flag.UintVar(&ControllerPort, "controllerPort", 8082, "the port of remote server controller")
	flag.UintVar(&TransferPort, "transferPort", 8083, "the port of remote server transfer")
	flag.Parse()

	var (
		controllerAddr = fmt.Sprintf("%s:%d", RemoteIP, ControllerPort)
		transferAddr   = fmt.Sprintf("%s:%d", RemoteIP, TransferPort)
		localAddr      = fmt.Sprintf("%s:%d", localServiceIP, localServicePort)
	)
	// connect to remote controller
	controllerConn, err := conn.ConnectServer(controllerAddr)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("connect to remote server controller successfully")

	reader := bufio.NewReader(controllerConn)

	for {
		data, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				log.Println("remote server controller EOF")
				return
			}
			log.Printf("read data from remote server controller,err:%s", err.Error())
			continue
		}
		if data == conn.NewConnection+"\n" {
			key, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					log.Println("remote server controller EOF")
					return
				}
				log.Printf("read data from remote server controller,err:%s", err.Error())
				continue
			}
			go pipe(key, transferAddr, localAddr)
		}

	}
}

func pipe(key string, transferAddr string, localAddr string) {
	transferConn, err := conn.ConnectServer(transferAddr)
	if err != nil {
		log.Printf("connect to transfer server,err:%s", err.Error())
		return
	}
	_, err = transferConn.Write([]byte(key))
	if err != nil {
		log.Printf("write key to transferConn,err:%s", err.Error())
		return
	}
	localConn, err := conn.ConnectServer(localAddr)
	if err != nil {
		log.Printf("connect to local service,err:%s", err.Error())
		transferConn.Close()
		return
	}
	conn.Pipe2Conn(transferConn, localConn)
}

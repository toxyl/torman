package loadblancer

import (
	"io"
	"net"
	"sync"
)

var (
	torInstances = []string{}
	torMutex     sync.Mutex
	currentTor   int
)

func Start(addr string, instances ...string) error {
	torInstances = instances
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	torMutex.Lock()
	torAddr := torInstances[currentTor]
	currentTor = (currentTor + 1) % len(torInstances)
	torMutex.Unlock()

	torConn, err := net.Dial("tcp", torAddr)
	if err != nil {
		return
	}
	defer torConn.Close()

	go func() {
		_, _ = io.Copy(torConn, clientConn)
	}()

	_, _ = io.Copy(clientConn, torConn)
}

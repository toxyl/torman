package loadblancer

import (
	"io"
	"net"
	"sync"
)

var (
	all     = []string{}
	mu      sync.Mutex
	current int
)

func Start(addr string, instances ...string) error {
	all = instances
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

func handleConnection(conn net.Conn) {
	defer conn.Close()

	mu.Lock()
	addr := all[current]
	current = (current + 1) % len(all)
	mu.Unlock()

	torConn, err := net.Dial("tcp", addr)
	if err != nil {
		return
	}
	defer torConn.Close()

	go func() {
		_, _ = io.Copy(torConn, conn)
	}()

	_, _ = io.Copy(conn, torConn)
}

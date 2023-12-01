package pine

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type TCPServer struct {
	addr string
	mu   sync.Mutex

	closed     bool
	listener   net.Listener
	activeConn map[net.Conn]struct{}

	messages []string
}

func NewTCPServer(addr string) *TCPServer {
	return &TCPServer{addr: addr, activeConn: map[net.Conn]struct{}{}}
}

func (t *TCPServer) Addr() string {
	return t.listener.Addr().String()
}

func (t *TCPServer) Run() (err error) {
	t.listener, err = net.Listen("tcp", t.addr)
	if err != nil {
		return err
	}
	go func() {
		for {
			conn, err := t.listener.Accept()
			if err != nil {
				if t.closed {
					return
				}
				log.Println(errors.Wrap(err, "Server: error accepting connection"))
				break
			}
			t.trackConnection(conn, true)
			go t.serve(conn)
		}
	}()
	return
}

func (t *TCPServer) Close() error {
	t.close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	ticker := time.NewTicker(time.Millisecond * 100)
	defer ticker.Stop()

	err := t.listener.Close()
	for {
		if t.numConnections() == 0 {
			break
		}

		select {
		case <-ctx.Done():
			return err
		case <-ticker.C:
		}
	}
	return err
}

func (t *TCPServer) numConnections() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.activeConn)
}
func (t *TCPServer) close() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closed = true
}

func (t *TCPServer) trackConnection(conn net.Conn, add bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if add {
		t.activeConn[conn] = struct{}{}
	} else {
		delete(t.activeConn, conn)
	}
}

func (t *TCPServer) serve(conn net.Conn) {
	defer func() {
		t.trackConnection(conn, false)
	}()
	defer conn.Close()

	tmp := make([]byte, 1024)
	data := make([]byte, 0)

	for {
		// read to the tmp var
		n, err := conn.Read(tmp)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Server: Read error - %s\n", err)
			}
			break
		}

		// append read data to full data
		data = append(data, tmp[:n]...)

		for {
			hasMessage, start, end := detectMessage(data)
			if !hasMessage {
				break
			}
			t.messages = append(t.messages, string(data[start:end-1]))
			data = data[end+1:]
		}
	}
}

func detectMessage(data []byte) (hasMessage bool, first int, last int) {
	hasMessage = false

	for i := range data {
		if i+1 < len(data) && data[i] == '\n' && data[i+1] == byte(0) {
			hasMessage = true
			first = 0
			last = i + 1
			return
		}
	}
	return
}

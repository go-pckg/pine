package gelf

import (
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	DefaultMaxReconnect   = 3
	DefaultReconnectDelay = 1
)

type TCPWriter struct {
	addr           string
	conn           net.Conn
	proto          string
	close          sync.Once
	mu             sync.Mutex
	MaxReconnect   int
	ReconnectDelay time.Duration
}

func NewTCPWriter(addr string) *TCPWriter {
	w := new(TCPWriter)
	w.MaxReconnect = DefaultMaxReconnect
	w.ReconnectDelay = DefaultReconnectDelay
	w.proto = "tcp"
	w.addr = addr
	return w
}

func (w *TCPWriter) connect() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.conn != nil {
		return nil
	}
	conn, err := net.Dial("tcp", w.addr)
	if err != nil {
		return err
	}
	w.conn = conn
	return nil
}

func (w *TCPWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.conn == nil {
		return nil
	}
	var err error
	w.close.Do(func() {
		err = w.conn.Close()
	})
	return err
}

func (w *TCPWriter) Write(p []byte) (n int, err error) {
	if errCon := w.connect(); errCon != nil {
		return 0, errCon
	}

	n, err = w.writeToSocketWithReconnectAttempts(p)
	if err != nil {
		return n, err
	}
	if n != len(p) {
		return n, fmt.Errorf("bad write (%d/%d)", n, len(p))
	}
	return n, nil
}

func (w *TCPWriter) writeToSocketWithReconnectAttempts(zBytes []byte) (n int, err error) {
	var errConn error
	var i int

	w.mu.Lock()
	defer w.mu.Unlock()
	for i = 0; i <= w.MaxReconnect; i++ {
		errConn = nil

		if w.conn != nil {
			n, err = w.conn.Write(zBytes)
		} else {
			err = fmt.Errorf("Connection was nil, will attempt reconnect")
		}
		if err != nil {
			time.Sleep(w.ReconnectDelay * time.Second)
			w.conn, errConn = net.Dial("tcp", w.addr)
		} else {
			break
		}
	}

	if i > w.MaxReconnect {
		return 0, fmt.Errorf("Maximum reconnection attempts was reached; giving up")
	}
	if errConn != nil {
		return 0, fmt.Errorf("Write Failed: %s\nReconnection failed: %s", err, errConn)
	}
	return n, nil
}

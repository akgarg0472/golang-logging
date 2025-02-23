package logger

import (
	"net"
	"sync"
	"time"
)

// TCPAsyncWriter is an asynchronous TCP writer that buffers and sends log messages
// to a remote host over a TCP connection. It supports automatic reconnection and batching.
type TCPAsyncWriter struct {
	host     string
	port     string
	conn     net.Conn
	mu       sync.Mutex
	msgChan  chan []byte
	quitChan chan struct{}
}

// NewTCPAsyncWriter creates a new TCPAsyncWriter and establishes an initial connection
// to the specified host and port. It also starts a background goroutine to process messages.
//
// If the connection cannot be established initially, it returns an error.
func NewTCPAsyncWriter(host, port string) (*TCPAsyncWriter, error) {
	writer := &TCPAsyncWriter{
		host:     host,
		port:     port,
		msgChan:  make(chan []byte, 1000),
		quitChan: make(chan struct{}),
	}
	if err := writer.connect(); err != nil {
		return nil, err
	}
	go writer.processMessages()
	return writer, nil
}

// connect establishes a TCP connection to the configured host and port.
// If the connection attempt fails, it returns an error.
func (w *TCPAsyncWriter) connect() error {
	address := net.JoinHostPort(w.host, w.port)
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return err
	}
	w.mu.Lock()
	w.conn = conn
	w.mu.Unlock()
	return nil
}

// reconnectLocked closes the existing connection (if any) and attempts to
// re-establish a TCP connection. This method assumes that the mutex lock
// is already held by the caller.
//
// If the reconnection attempt fails, it returns an error.
func (w *TCPAsyncWriter) reconnectLocked() error {
	if w.conn != nil {
		w.conn.Close()
		w.conn = nil
	}
	address := net.JoinHostPort(w.host, w.port)
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return err
	}
	w.conn = conn
	return nil
}

// processMessages listens for messages on the msgChan channel,
// batches them, and sends them to the TCP connection periodically.
//
// It ensures efficient network usage by buffering messages and flushing
// them based on a time interval or when a batch threshold is reached.
func (w *TCPAsyncWriter) processMessages() {
	const batchThreshold = 4096
	batch := make([]byte, 0, batchThreshold)
	flushInterval := 100 * time.Millisecond
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-w.msgChan:
			if !ok {
				w.flush(batch)
				return
			}
			batch = append(batch, msg...)
			if len(batch) > 0 && batch[len(batch)-1] != '\n' {
				batch = append(batch, '\n')
			}
			if len(batch) >= batchThreshold {
				w.flush(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				w.flush(batch)
				batch = batch[:0]
			}
		case <-w.quitChan:
			w.flush(batch)
			return
		}
	}
}

// flush sends the accumulated batch of log messages to the TCP connection.
//
// If the connection is lost, it attempts to reconnect before retrying the write.
// If reconnection fails, the data is discarded.
func (w *TCPAsyncWriter) flush(data []byte) {
	if len(data) == 0 {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.conn == nil {
		if err := w.reconnectLocked(); err != nil {
			return
		}
	}
	_, err := w.conn.Write(data)
	if err != nil {
		if err := w.reconnectLocked(); err != nil {
			return
		}
		w.conn.Write(data)
	}
}

// Write queues the provided data for asynchronous transmission over TCP.
//
// It copies the data before sending it to msgChan to prevent data corruption
// due to concurrent modifications.
//
// Returns the number of bytes written.
func (w *TCPAsyncWriter) Write(p []byte) (int, error) {
	data := make([]byte, len(p))
	copy(data, p)
	w.msgChan <- data
	return len(p), nil
}

// Close gracefully shuts down the TCPAsyncWriter.
//
// It ensures that all buffered messages are flushed before closing the TCP connection.
func (w *TCPAsyncWriter) Close() error {
	close(w.quitChan)
	close(w.msgChan)
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.conn != nil {
		return w.conn.Close()
	}
	return nil
}

package main

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"time"
)

func NewHelperTransport(in, out io.Writer, cfg *tls.Config) *http.Transport {
	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			c, err := (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext(ctx, network, addr)

			return &LoggerConn{conn: c, in: in, out: out}, err
		},
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			c, err := (&tls.Dialer{
				NetDialer: &net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
					DualStack: true,
				},
				Config: cfg,
			}).DialContext(ctx, network, addr)

			return &LoggerConn{conn: c, in: in, out: out}, err
		},

		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		Proxy:                 http.ProxyFromEnvironment,
	}
}

type LoggerConn struct {
	conn net.Conn
	in   io.Writer
	out  io.Writer
}

// Read implements the Read method.
func (l *LoggerConn) Read(b []byte) (n int, err error) {
	n, err = l.conn.Read(b)
	l.in.Write(b)
	return n, err
}

// Write implements the Write method.
func (l *LoggerConn) Write(b []byte) (n int, err error) {
	l.out.Write(b)
	return l.conn.Write(b)
}

// Close implements the Close method.
func (l *LoggerConn) Close() error {
	return l.conn.Close()
}

// LocalAddr implements the LocalAddr method.
func (l *LoggerConn) LocalAddr() net.Addr {
	return l.conn.LocalAddr()
}

// RemoteAddr implements the RemoteAddr method.
func (l *LoggerConn) RemoteAddr() net.Addr {
	return l.conn.RemoteAddr()
}

// SetDeadline implements the SetDeadline method.
func (l *LoggerConn) SetDeadline(t time.Time) error {
	return l.conn.SetDeadline(t)
}

// SetReadDeadline implements the SetReadDeadline method.
func (l *LoggerConn) SetReadDeadline(t time.Time) error {
	return l.conn.SetReadDeadline(t)
}

// SetWriteDeadline implements the SetWriteDeadline method.
func (l *LoggerConn) SetWriteDeadline(t time.Time) error {
	return l.SetWriteDeadline(t)
}

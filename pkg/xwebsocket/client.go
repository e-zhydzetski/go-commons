package xwebsocket

import (
	"context"
	"crypto/tls"
	"errors"
	"github.com/gobwas/ws"
	"time"
)

type ClientOption func(dialer *ws.Dialer)

func ClientTLSConfig(tc *tls.Config) ClientOption {
	return func(dialer *ws.Dialer) {
		dialer.TLSConfig = tc
	}
}

func NewClient(ctx context.Context, connectAddr string, sessionFactoryFunc WSSessionFactoryFunc, opts ...ClientOption) (WSSession, error) {
	dialer := ws.Dialer{}
	for _, opt := range opts {
		opt(&dialer)
	}
	conn, bfr, _, err := dialer.Dial(ctx, connectAddr)
	if err != nil {
		return nil, err
	}
	if bfr != nil {
		// https://github.com/gobwas/ws/issues/19, may happens only if server send data immediately after handshake
		_ = conn.Close()
		return nil, errors.New("some frame data got in handshake buffer")
	}

	sess := sessionFactoryFunc(ctx, connWithTimeout{
		Conn: conn,
		wt:   1 * time.Second, // TODO make configurable
		rt:   0,               // can't use read timeout in wait model (without events)
	})

	closedCh := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			sess.Close(ctx.Err())
		case <-closedCh:
		}
	}()

	go func() {
		for {
			if err := sess.Consume(); err != nil {
				sess.Close(err)
				break
			}
		}
		close(closedCh)
	}()
	return sess, err
}

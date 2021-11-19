package xwebsocket

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/gobwas/ws"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	addr *net.TCPAddr
}

func (s Server) Port() int {
	return s.addr.Port
}

func StartSimpleServer(ctx context.Context, g *errgroup.Group, addr string, sessionFactoryFunc WSSessionFactoryFunc) (*Server, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	g.Go(func() error {
		// opened sessions monitoring to cleanup after ctx.Done
		var sessionIDSeq uint64
		var mx sync.Mutex
		sessions := map[uint64]WSSession{}
		defer func() { // will be executed synchronously after accept loop, so no new sessions will be added in parallel or later
			mx.Lock()
			for _, session := range sessions {
				session.Close(ctx.Err())
			}
			sessions = nil // safe, as no inserts will be later, and delete from nil map is ok
			mx.Unlock()
		}()

		for {
			conn, err := ln.Accept()
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Temporary() {
					continue
				}
				return err
			}

			_, err = ws.Upgrade(conn)
			if err != nil {
				_ = conn.Close()
				continue
			}

			sessionIDSeq++ // unique ID of the session
			sess := sessionFactoryFunc(ctx, connWithTimeout{
				Conn: conn,
				wt:   1 * time.Second, // TODO make configurable
				rt:   0,               // can't use read timeout in wait model (without events)
			})
			mx.Lock()
			sessions[sessionIDSeq] = sess
			mx.Unlock()

			go func(sid uint64) {
				for {
					if err := sess.Consume(); err != nil {
						sess.Close(err)
						break
					}
				}
				mx.Lock()
				delete(sessions, sid) // session closed
				mx.Unlock()
			}(sessionIDSeq)
		}
	})

	return &Server{
		addr: ln.Addr().(*net.TCPAddr),
	}, nil
}

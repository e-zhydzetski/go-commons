package xhttp

import (
	"context"
	"errors"
	"golang.org/x/sync/errgroup"
	"net"
	"net/http"
)

type Server struct {
	addrCh chan *net.TCPAddr
	addr   *net.TCPAddr
}

func (s *Server) ListeningPort() (int, error) {
	if s.addr == nil {
		s.addr = <-s.addrCh
	}
	if s.addr == nil {
		return 0, errors.New("server not listening")
	}
	return s.addr.Port, nil
}

func StartServer(ctx context.Context, g *errgroup.Group, addr string, handler http.Handler) *Server {
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}
	server.BaseContext = func(_ net.Listener) context.Context {
		return ctx
	}
	g.Go(func() error {
		<-ctx.Done()
		return server.Shutdown(context.Background()) // wait forever for live connections, maybe add timeout
	})
	addrCh := make(chan *net.TCPAddr, 1)
	g.Go(func() error { // modified part of server.ListenAndServer
		defer close(addrCh)
		ln, err := net.Listen("tcp", server.Addr)
		if err != nil {
			return err
		}
		addrCh <- ln.Addr().(*net.TCPAddr)
		return server.Serve(ln)
	})

	return &Server{
		addrCh: addrCh,
	}
}

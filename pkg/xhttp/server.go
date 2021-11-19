package xhttp

import (
	"context"
	"golang.org/x/sync/errgroup"
	"net"
	"net/http"
)

type Server struct {
	addr *net.TCPAddr
}

func (s Server) Port() int {
	return s.addr.Port
}

func StartServer(ctx context.Context, g *errgroup.Group, addr string, handler http.Handler) (*Server, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

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
	g.Go(func() error { // modified part of server.ListenAndServer
		return server.Serve(ln)
	})

	return &Server{
		addr: ln.Addr().(*net.TCPAddr),
	}, nil
}

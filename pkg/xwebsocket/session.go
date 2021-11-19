package xwebsocket

import (
	"context"
	"net"
)

type WSSession interface {
	Consume() error
	Close(err error) // should be idempotent
}

type WSSessionFactoryFunc func(ctx context.Context, conn net.Conn) WSSession

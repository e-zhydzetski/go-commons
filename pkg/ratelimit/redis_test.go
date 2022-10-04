package ratelimit

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gotest.tools/assert"
	"math/rand"
	"testing"
	"time"
)

func TestRedisStrategy(t *testing.T) {
	ctx := context.Background()
	client, terminate, err := startRedis(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer terminate()

	strategy := NewRedisStrategy(client, time.Now)

	req := &Request{
		Key:      "test",
		Limit:    50,
		Duration: time.Second,
	}

	// perform concurrent requests with random short delay, report about time of allowed requests by allowed channel
	stop := make(chan struct{})
	allowed := make(chan time.Time)
	for i := 0; i < 10; i++ {
		go func() {
			for {
				select {
				case <-stop:
					return
				case <-time.After(time.Millisecond * time.Duration(rand.Intn(100))):
					rt := time.Now()
					res, err := strategy.Run(ctx, req)
					assert.NilError(t, err)
					if res.State == Allow {
						allowed <- rt
					}
				}
			}
		}()
	}

	// check that allowed requests are inside rate limit floating window
	func() {
		defer close(stop)

		ctx, cancel := context.WithTimeout(ctx, time.Second*5) // test duration
		defer cancel()

		var reqs []time.Time
		for {
			select {
			case <-ctx.Done():
				return
			case rt := <-allowed:
				t.Log(rt)
				reqs = append(reqs, rt)
				min := rt.Add(-req.Duration)
				l := -1
				for i, r := range reqs {
					if r.Before(min) {
						l = i
					} else {
						break
					}
				}
				if l+1 < len(reqs) {
					reqs = reqs[l+1:]
				} else {
					reqs = nil
				}

				if len(reqs) > int(req.Limit) {
					t.Fatal("limit exceeded")
				}
			}
		}
	}()
}

func startRedis(ctx context.Context) (client *redis.Client, terminate func(), err error) {
	req := testcontainers.ContainerRequest{
		Image:        "redis:6.2.6-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("* Ready to accept connections"),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return
	}
	terminate = func() {
		_ = container.Terminate(ctx)
	}
	defer func() {
		if err != nil {
			terminate()
		}
	}()
	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		return
	}
	host, err := container.Host(ctx)
	if err != nil {
		return
	}
	uri := fmt.Sprintf("redis://%s:%s", host, port.Port())

	options, err := redis.ParseURL(uri)
	if err != nil {
		return
	}
	client = redis.NewClient(options)
	return
}

package ratelimit

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"strconv"
	"time"
)

func NewRedisStrategy(client *redis.Client, now func() time.Time) Strategy {
	return &redisStrategy{
		client: client,
		now:    now,
	}
}

type redisStrategy struct {
	client *redis.Client
	now    func() time.Time
}

func (s *redisStrategy) Run(ctx context.Context, r *Request) (*Result, error) {
	now := s.now()
	expiresAt := now.Add(r.Duration)
	minimum := now.Add(-r.Duration)
	minimumStr := strconv.FormatInt(minimum.UnixMilli(), 10)

	result, err := s.client.ZCount(ctx, r.Key, minimumStr, "+inf").Uint64()
	if err == nil && result >= r.Limit {
		return &Result{
			State:         Deny,
			TotalRequests: result,
			ExpiresAt:     expiresAt,
		}, nil
	}

	item := uuid.New()

	p := s.client.Pipeline()

	removeByScore := p.ZRemRangeByScore(ctx, r.Key, "0", minimumStr)

	add := p.ZAdd(ctx, r.Key, &redis.Z{
		Score:  float64(now.UnixMilli()),
		Member: item.String(),
	})

	count := p.ZCount(ctx, r.Key, "-inf", "+inf")

	if _, err := p.Exec(ctx); err != nil {
		return nil, fmt.Errorf("failed to execute sorted set pipeline for key %v: %v", r.Key, err)
	}

	if err := removeByScore.Err(); err != nil {
		return nil, fmt.Errorf("failed to remove items from key %v: %v", r.Key, err)
	}

	if err := add.Err(); err != nil {
		return nil, fmt.Errorf("failed to add item to key %v: %v", r.Key, err)
	}

	totalRequests, err := count.Result()
	if err != nil {
		return nil, fmt.Errorf("failed to count items for key %v: %v", r.Key, err)
	}

	requests := uint64(totalRequests)

	if requests > r.Limit {
		return &Result{
			State:         Deny,
			TotalRequests: requests,
			ExpiresAt:     expiresAt,
		}, nil
	}

	return &Result{
		State:         Allow,
		TotalRequests: requests,
		ExpiresAt:     expiresAt,
	}, nil
}

package store

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yourname/matchmaker-lite/pkg/types"
)

type Store interface {
	Enqueue(ctx context.Context, req types.JoinRequest) (string, error)
	Dequeue(ctx context.Context, playerID string) error
	PeekQueue(ctx context.Context, n int) ([]types.Player, error)
	CommitMatch(ctx context.Context, players []types.Player) error
	Close() error
}

type RedisStore struct{ rdb *redis.Client }

const (
	queueKey = "mm:queue" // ZSET: score=MMR, member=playerID
	metaKey  = "mm:meta:" // HASH per player: mmr, joined_at
)

func NewRedisStore(addr, password string) *RedisStore {
	return &RedisStore{rdb: redis.NewClient(&redis.Options{Addr: addr, Password: password})}
}

func (s *RedisStore) Close() error { return s.rdb.Close() }

func (s *RedisStore) Enqueue(ctx context.Context, req types.JoinRequest) (string, error) {
	pipe := s.rdb.TxPipeline()
	pipe.ZAdd(ctx, queueKey, redis.Z{Score: req.MMR, Member: req.PlayerID})
	pipe.HSet(ctx, metaKey+req.PlayerID, map[string]any{"mmr": req.MMR, "joined_at": time.Now().Unix()})
	_, err := pipe.Exec(ctx)
	return req.PlayerID, err
}

func (s *RedisStore) Dequeue(ctx context.Context, playerID string) error {
	pipe := s.rdb.TxPipeline()
	pipe.ZRem(ctx, queueKey, playerID)
	pipe.Del(ctx, metaKey+playerID)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *RedisStore) PeekQueue(ctx context.Context, n int) ([]types.Player, error) {
	ids, err := s.rdb.ZRange(ctx, queueKey, 0, int64(n-1)).Result()
	if err != nil {
		return nil, err
	}
	res := make([]types.Player, 0, len(ids))
	for _, id := range ids {
		mmr, err := s.rdb.HGet(ctx, metaKey+id, "mmr").Float64()
		if err != nil {
			return nil, err
		}
		res = append(res, types.Player{PlayerID: id, MMR: mmr})
	}
	return res, nil
}

func (s *RedisStore) CommitMatch(ctx context.Context, players []types.Player) error {
	pipe := s.rdb.TxPipeline()
	for _, p := range players {
		pipe.ZRem(ctx, queueKey, p.PlayerID)
		pipe.Del(ctx, metaKey+p.PlayerID)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("commit match: %w", err)
	}
	return nil
}

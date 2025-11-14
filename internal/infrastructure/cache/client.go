package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"caatsm/internal/config"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	client *redis.Client
	enabled bool
}

func NewClient(cfg *config.RedisConfig) (*Client, error) {
	if !cfg.Enabled {
		return &Client{enabled: false}, nil
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Client{
		client:  rdb,
		enabled: true,
	}, nil
}

// Get retrieves a value from cache
func (c *Client) Get(ctx context.Context, key string, dest interface{}) error {
	if !c.enabled {
		return fmt.Errorf("cache is disabled")
	}

	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return fmt.Errorf("key not found")
	}
	if err != nil {
		return fmt.Errorf("failed to get from cache: %w", err)
	}

	return json.Unmarshal([]byte(val), dest)
}

// Set stores a value in cache
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if !c.enabled {
		return fmt.Errorf("cache is disabled")
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	return c.client.Set(ctx, key, data, expiration).Err()
}

// Delete removes a key from cache
func (c *Client) Delete(ctx context.Context, key string) error {
	if !c.enabled {
		return nil
	}

	return c.client.Del(ctx, key).Err()
}

// Close closes the Redis connection
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// IsEnabled returns whether caching is enabled
func (c *Client) IsEnabled() bool {
	return c.enabled
}


// pkg/redis/health.go
package redis

import "context"

func (c *Client) HealthCheck(ctx context.Context) error {
	return c.Ping(ctx).Err()
}

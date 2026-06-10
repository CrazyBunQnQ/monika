package redis

import (
	"context"
	"fmt"
	"strings"

	"monika/pkg/dbdriver"

	"github.com/redis/go-redis/v9"
)

var readOnlyCommands = map[string]bool{
	"GET": true, "MGET": true, "TYPE": true, "SCAN": true,
	"HGET": true, "HGETALL": true, "LRANGE": true, "SMEMBERS": true,
	"ZCARD": true, "ZSCORE": true, "ZRANGE": true, "SCARD": true,
	"SISMEMBER": true, "EXISTS": true, "TTL": true, "STRLEN": true,
	"INFO": true, "DBSIZE": true,
}

func init() {
	dbdriver.Register(&Driver{})
}

type Driver struct{}

func (d *Driver) ID() string { return "redis" }

func (d *Driver) Open(dsn string) (dbdriver.Connection, error) {
	opts, err := redis.ParseURL(dsn)
	if err != nil {
		return nil, fmt.Errorf("redis: parse dsn: %w", err)
	}
	client := redis.NewClient(opts)
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("redis: ping: %w", err)
	}
	return &Conn{client: client}, nil
}

type Conn struct {
	client *redis.Client
}

func (c *Conn) Query(ctx context.Context, query string) (*dbdriver.QueryResult, error) {
	parts := strings.Fields(query)
	if len(parts) == 0 {
		return nil, fmt.Errorf("redis: empty query")
	}
	cmd := strings.ToUpper(parts[0])
	if !readOnlyCommands[cmd] {
		return nil, fmt.Errorf("redis: command %q is not allowed (read-only mode)", cmd)
	}

	args := make([]any, len(parts[1:]))
	for i, p := range parts[1:] {
		args[i] = p
	}
	cmdArgs := make([]any, 0, len(parts))
	cmdArgs = append(cmdArgs, cmd)
	cmdArgs = append(cmdArgs, args...)
	val, err := c.client.Do(ctx, cmdArgs...).Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("redis: %s: %w", cmd, err)
	}

	result := &dbdriver.QueryResult{
		Columns: []string{cmd},
		Rows:    [][]any{{formatResult(val)}},
		Tag:     cmd,
	}
	return result, nil
}

func (c *Conn) Schema(ctx context.Context, filter string) (*dbdriver.SchemaResult, error) {
	info, err := c.client.Info(ctx, "keyspace").Result()
	if err != nil {
		return nil, fmt.Errorf("redis: info keyspace: %w", err)
	}

	var tables []dbdriver.TableInfo
	var cursor uint64
	for {
		var keys []string
		keys, cursor, err = c.client.Scan(ctx, cursor, "*", 100).Result()
		if err != nil {
			return nil, fmt.Errorf("redis: scan: %w", err)
		}

		for _, key := range keys {
			if filter != "" && !strings.Contains(key, filter) {
				continue
			}
			keyType, err := c.client.Type(ctx, key).Result()
			if err != nil {
				continue
			}
			tables = append(tables, dbdriver.TableInfo{
				Name: key,
				Columns: []dbdriver.ColumnInfo{
					{Name: "key", Type: "string", Nullable: false},
					{Name: "type", Type: keyType, Nullable: false},
					{Name: "ttl", Type: "integer", Nullable: true},
				},
			})
		}
		if cursor == 0 {
			break
		}
	}

	_ = info
	return &dbdriver.SchemaResult{Tables: tables}, nil
}

func (c *Conn) Close() error {
	return c.client.Close()
}

func formatResult(v any) any {
	switch val := v.(type) {
	case []any:
		s := make([]string, len(val))
		for i, item := range val {
			s[i] = fmt.Sprintf("%v", item)
		}
		return strings.Join(s, ", ")
	default:
		return fmt.Sprintf("%v", val)
	}
}

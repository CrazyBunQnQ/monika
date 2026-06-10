package mongo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"monika/pkg/dbdriver"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func init() {
	dbdriver.Register(&Driver{})
}

type Driver struct{}

func (d *Driver) ID() string { return "mongo" }

func (d *Driver) Open(dsn string) (dbdriver.Connection, error) {
	ctx := context.Background()
	client, err := mongo.Connect(options.Client().ApplyURI(dsn))
	if err != nil {
		return nil, fmt.Errorf("mongo: connect: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		client.Disconnect(ctx)
		return nil, fmt.Errorf("mongo: ping: %w", err)
	}
	return &Conn{client: client}, nil
}

type Conn struct {
	client *mongo.Client
}

type findQuery struct {
	Collection string         `json:"collection"`
	Filter     map[string]any `json:"filter"`
	Limit      int64          `json:"limit"`
}

func (c *Conn) Query(ctx context.Context, query string) (*dbdriver.QueryResult, error) {
	var fq findQuery
	if err := json.Unmarshal([]byte(query), &fq); err != nil {
		return nil, fmt.Errorf("mongo: parse query: %w", err)
	}
	if fq.Collection == "" {
		return nil, fmt.Errorf("mongo: collection is required")
	}

	dbName := c.getDBName()
	coll := c.client.Database(dbName).Collection(fq.Collection)

	filter := bson.M(fq.Filter)
	if filter == nil {
		filter = bson.M{}
	}

	opts := options.Find()
	if fq.Limit > 0 {
		opts.SetLimit(fq.Limit)
	}

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("mongo: find: %w", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("mongo: cursor all: %w", err)
	}

	colSeen := make(map[string]bool)
	var cols []string
	for _, doc := range results {
		for k := range doc {
			if !colSeen[k] {
				colSeen[k] = true
				cols = append(cols, k)
			}
		}
	}

	rows := make([][]any, len(results))
	for i, doc := range results {
		ordered := make([]any, len(cols))
		for j, colName := range cols {
			ordered[j] = formatBSON(doc[colName])
		}
		rows[i] = ordered
	}

	return &dbdriver.QueryResult{
		Columns: cols,
		Rows:    rows,
		Tag:     "FIND",
	}, nil
}

func (c *Conn) Schema(ctx context.Context, filter string) (*dbdriver.SchemaResult, error) {
	dbName := c.getDBName()
	db := c.client.Database(dbName)

	names, err := db.ListCollectionNames(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("mongo: list collections: %w", err)
	}

	var tables []dbdriver.TableInfo
	for _, name := range names {
		if filter != "" && !strings.Contains(name, filter) {
			continue
		}
		coll := db.Collection(name)
		var sample bson.M
		err := coll.FindOne(ctx, bson.M{}).Decode(&sample)
		if err != nil {
			tables = append(tables, dbdriver.TableInfo{Name: name})
			continue
		}

		var cols []dbdriver.ColumnInfo
		for k, v := range sample {
			cols = append(cols, dbdriver.ColumnInfo{
				Name: k,
				Type: inferType(v),
			})
		}
		tables = append(tables, dbdriver.TableInfo{Name: name, Columns: cols})
	}

	return &dbdriver.SchemaResult{Tables: tables}, nil
}

func (c *Conn) Close() error {
	return c.client.Disconnect(context.Background())
}

func (c *Conn) getDBName() string {
	return c.client.Database("").Name()
}

func formatBSON(v any) any {
	switch val := v.(type) {
	case bson.ObjectID:
		return val.Hex()
	case bson.M, bson.A:
		b, _ := json.Marshal(val)
		return string(b)
	default:
		return val
	}
}

func inferType(v any) string {
	switch v.(type) {
	case string:
		return "string"
	case int, int32, int64:
		return "integer"
	case float64:
		return "double"
	case bool:
		return "boolean"
	case bson.ObjectID:
		return "objectId"
	case bson.M, bson.A:
		return "object"
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%T", v)
	}
}

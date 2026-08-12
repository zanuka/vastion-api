package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

type Client struct {
	inner *mongo.Client
}

func Connect(uri string) (*Client, error) {
	inner, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	return &Client{inner: inner}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	return c.inner.Ping(ctx, readpref.Primary())
}

func (c *Client) Disconnect(ctx context.Context) error {
	if c == nil || c.inner == nil {
		return nil
	}
	return c.inner.Disconnect(ctx)
}

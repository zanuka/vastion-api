package mongodb

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

const (
	collectionSites            = "sites"
	collectionSensors          = "sensors"
	collectionDetections       = "detections"
	collectionAcknowledgements = "acknowledgements"

	defaultDatabase = "watchdesk"
)

type Client struct {
	inner *mongo.Client
	db    *mongo.Database
}

func Connect(uri string) (*Client, error) {
	name, err := databaseName(uri)
	if err != nil {
		return nil, err
	}
	inner, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	return &Client{inner: inner, db: inner.Database(name)}, nil
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

func (c *Client) ResetCollections(ctx context.Context) error {
	names := []string{
		collectionAcknowledgements,
		collectionDetections,
		collectionSensors,
		collectionSites,
	}
	for _, name := range names {
		if _, err := c.db.Collection(name).DeleteMany(ctx, bson.D{}); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) EnsureIndexes(ctx context.Context) error {
	if err := createIndexes(ctx, c.db.Collection(collectionSites), siteIndexModels()); err != nil {
		return fmt.Errorf("sites indexes: %w", err)
	}
	if err := createIndexes(ctx, c.db.Collection(collectionSensors), sensorIndexModels()); err != nil {
		return fmt.Errorf("sensors indexes: %w", err)
	}
	if err := createIndexes(ctx, c.db.Collection(collectionDetections), detectionIndexModels()); err != nil {
		return fmt.Errorf("detections indexes: %w", err)
	}
	if err := createIndexes(ctx, c.db.Collection(collectionAcknowledgements), acknowledgementIndexModels()); err != nil {
		return fmt.Errorf("acknowledgements indexes: %w", err)
	}
	return nil
}

func createIndexes(ctx context.Context, coll *mongo.Collection, models []mongo.IndexModel) error {
	if len(models) == 0 {
		return nil
	}
	_, err := coll.Indexes().CreateMany(ctx, models)
	return err
}

func siteIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "code", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("code_unique"),
		},
	}
}

func sensorIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "siteId", Value: 1}},
			Options: options.Index().SetName("siteId"),
		},
		{
			Keys: bson.D{
				{Key: "siteId", Value: 1},
				{Key: "code", Value: 1},
			},
			Options: options.Index().SetUnique(true).SetName("siteId_code_unique"),
		},
	}
}

func detectionIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "siteId", Value: 1},
				{Key: "status", Value: 1},
				{Key: "detectedAt", Value: -1},
			},
			Options: options.Index().SetName("siteId_status_detectedAt"),
		},
		{
			Keys:    bson.D{{Key: "sensorId", Value: 1}},
			Options: options.Index().SetName("sensorId"),
		},
	}
}

func acknowledgementIndexModels() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "detectionId", Value: 1},
				{Key: "createdAt", Value: -1},
			},
			Options: options.Index().SetName("detectionId_createdAt"),
		},
	}
}

func databaseName(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	name := strings.Trim(u.Path, "/")
	if name == "" {
		return defaultDatabase, nil
	}
	return name, nil
}

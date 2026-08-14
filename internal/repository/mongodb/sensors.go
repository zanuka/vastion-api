package mongodb

import (
	"context"

	"github.com/zanuka/vastion-api/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var _ domain.SensorRepository = (*SensorRepository)(nil)

type SensorRepository struct {
	coll *mongo.Collection
}

func NewSensorRepository(c *Client) *SensorRepository {
	return &SensorRepository{coll: c.db.Collection(collectionSensors)}
}

type sensorDoc struct {
	ID     bson.ObjectID `bson:"_id"`
	SiteID bson.ObjectID `bson:"siteId"`
	Name   string        `bson:"name"`
	Code   string        `bson:"code"`
}

func (d sensorDoc) toDomain() domain.Sensor {
	return domain.Sensor{
		ID:     d.ID.Hex(),
		SiteID: d.SiteID.Hex(),
		Name:   d.Name,
		Code:   d.Code,
	}
}

func (r *SensorRepository) Insert(ctx context.Context, sensor *domain.Sensor) error {
	id, err := objectIDOrNew(sensor.ID)
	if err != nil {
		return err
	}
	siteID, err := parseObjectID(sensor.SiteID)
	if err != nil {
		return err
	}
	doc := sensorDoc{
		ID:     id,
		SiteID: siteID,
		Name:   sensor.Name,
		Code:   sensor.Code,
	}
	if _, err := r.coll.InsertOne(ctx, doc); err != nil {
		return err
	}
	sensor.ID = id.Hex()
	return nil
}

func (r *SensorRepository) GetByID(ctx context.Context, id string) (*domain.Sensor, error) {
	oid, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	var doc sensorDoc
	if err := r.coll.FindOne(ctx, bson.D{{Key: "_id", Value: oid}}).Decode(&doc); err != nil {
		return nil, mapFindErr(err)
	}
	sensor := doc.toDomain()
	return &sensor, nil
}

func (r *SensorRepository) ListBySiteID(ctx context.Context, siteID string) ([]domain.Sensor, error) {
	oid, err := parseObjectID(siteID)
	if err != nil {
		return nil, err
	}
	opts := options.Find().SetSort(bson.D{{Key: "code", Value: 1}})
	cur, err := r.coll.Find(ctx, bson.D{{Key: "siteId", Value: oid}}, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(ctx) }()

	var docs []sensorDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]domain.Sensor, 0, len(docs))
	for _, doc := range docs {
		out = append(out, doc.toDomain())
	}
	return out, nil
}

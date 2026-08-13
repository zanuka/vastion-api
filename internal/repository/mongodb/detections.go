package mongodb

import (
	"context"
	"time"

	"github.com/zanuka/vastion-api/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var _ domain.DetectionRepository = (*DetectionRepository)(nil)

type DetectionRepository struct {
	coll *mongo.Collection
}

func NewDetectionRepository(c *Client) *DetectionRepository {
	return &DetectionRepository{coll: c.db.Collection(collectionDetections)}
}

type detectionDoc struct {
	ID         bson.ObjectID `bson:"_id"`
	SiteID     bson.ObjectID `bson:"siteId"`
	SensorID   bson.ObjectID `bson:"sensorId"`
	Status     string        `bson:"status"`
	Severity   string        `bson:"severity"`
	Summary    string        `bson:"summary"`
	DetectedAt time.Time     `bson:"detectedAt"`
}

func (d detectionDoc) toDomain() domain.Detection {
	return domain.Detection{
		ID:         d.ID.Hex(),
		SiteID:     d.SiteID.Hex(),
		SensorID:   d.SensorID.Hex(),
		Status:     domain.DetectionStatus(d.Status),
		Severity:   domain.DetectionSeverity(d.Severity),
		Summary:    d.Summary,
		DetectedAt: d.DetectedAt.UTC(),
	}
}

func (r *DetectionRepository) Insert(ctx context.Context, detection *domain.Detection) error {
	id, err := objectIDOrNew(detection.ID)
	if err != nil {
		return err
	}
	siteID, err := parseObjectID(detection.SiteID)
	if err != nil {
		return err
	}
	sensorID, err := parseObjectID(detection.SensorID)
	if err != nil {
		return err
	}
	status := detection.Status
	if status == "" {
		status = domain.DetectionStatusOpen
	}
	doc := detectionDoc{
		ID:         id,
		SiteID:     siteID,
		SensorID:   sensorID,
		Status:     string(status),
		Severity:   string(detection.Severity),
		Summary:    detection.Summary,
		DetectedAt: detection.DetectedAt.UTC(),
	}
	if _, err := r.coll.InsertOne(ctx, doc); err != nil {
		return err
	}
	detection.ID = id.Hex()
	detection.Status = status
	return nil
}

func (r *DetectionRepository) GetByID(ctx context.Context, id string) (*domain.Detection, error) {
	oid, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	var doc detectionDoc
	if err := r.coll.FindOne(ctx, bson.D{{Key: "_id", Value: oid}}).Decode(&doc); err != nil {
		return nil, mapFindErr(err)
	}
	detection := doc.toDomain()
	return &detection, nil
}

func (r *DetectionRepository) List(ctx context.Context, filter domain.DetectionListFilter) ([]domain.Detection, error) {
	q, err := detectionQuery(filter)
	if err != nil {
		return nil, err
	}
	opts := options.Find().SetSort(bson.D{
		{Key: "detectedAt", Value: -1},
		{Key: "_id", Value: -1},
	})
	cur, err := r.coll.Find(ctx, q, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var docs []detectionDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]domain.Detection, 0, len(docs))
	for _, doc := range docs {
		out = append(out, doc.toDomain())
	}
	return out, nil
}

func detectionQuery(filter domain.DetectionListFilter) (bson.D, error) {
	q := bson.D{}
	if filter.SiteID != "" {
		id, err := parseObjectID(filter.SiteID)
		if err != nil {
			return nil, err
		}
		q = append(q, bson.E{Key: "siteId", Value: id})
	}
	if filter.SensorID != "" {
		id, err := parseObjectID(filter.SensorID)
		if err != nil {
			return nil, err
		}
		q = append(q, bson.E{Key: "sensorId", Value: id})
	}
	if filter.Status != "" {
		q = append(q, bson.E{Key: "status", Value: string(filter.Status)})
	}
	if filter.Severity != "" {
		q = append(q, bson.E{Key: "severity", Value: string(filter.Severity)})
	}
	return q, nil
}

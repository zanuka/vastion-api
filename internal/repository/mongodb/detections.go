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

type provenanceDoc struct {
	SensorID     string `bson:"sensorId"`
	SensorName   string `bson:"sensorName,omitempty"`
	Model        string `bson:"model,omitempty"`
	ModelVersion string `bson:"modelVersion,omitempty"`
}

type detectionDoc struct {
	ID          bson.ObjectID `bson:"_id"`
	SiteID      bson.ObjectID `bson:"siteId"`
	SensorID    bson.ObjectID `bson:"sensorId"`
	Status      string        `bson:"status"`
	Severity    string        `bson:"severity"`
	Confidence  float64       `bson:"confidence"`
	Summary     string        `bson:"summary"`
	DetectedAt  time.Time     `bson:"detectedAt"`
	LastUpdated time.Time     `bson:"lastUpdated"`
	Provenance  provenanceDoc `bson:"provenance"`
}

func (d detectionDoc) toDomain() domain.Detection {
	detectedAt := d.DetectedAt.UTC()
	lastUpdated := d.LastUpdated.UTC()
	if lastUpdated.IsZero() {
		lastUpdated = detectedAt
	}
	prov := domain.Provenance{
		SensorID:     d.Provenance.SensorID,
		SensorName:   d.Provenance.SensorName,
		Model:        d.Provenance.Model,
		ModelVersion: d.Provenance.ModelVersion,
	}
	if prov.SensorID == "" {
		prov.SensorID = d.SensorID.Hex()
	}
	return domain.Detection{
		ID:          d.ID.Hex(),
		SiteID:      d.SiteID.Hex(),
		SensorID:    d.SensorID.Hex(),
		Status:      domain.DetectionStatus(d.Status),
		Severity:    domain.DetectionSeverity(d.Severity),
		Confidence:  d.Confidence,
		Summary:     d.Summary,
		DetectedAt:  detectedAt,
		LastUpdated: lastUpdated,
		Provenance:  prov,
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
	detectedAt := detection.DetectedAt.UTC()
	lastUpdated := detection.LastUpdated.UTC()
	if lastUpdated.IsZero() {
		lastUpdated = detectedAt
	}
	prov := detection.Provenance
	if prov.SensorID == "" {
		prov.SensorID = sensorID.Hex()
	}
	doc := detectionDoc{
		ID:          id,
		SiteID:      siteID,
		SensorID:    sensorID,
		Status:      string(status),
		Severity:    string(detection.Severity),
		Confidence:  detection.Confidence,
		Summary:     detection.Summary,
		DetectedAt:  detectedAt,
		LastUpdated: lastUpdated,
		Provenance: provenanceDoc{
			SensorID:     prov.SensorID,
			SensorName:   prov.SensorName,
			Model:        prov.Model,
			ModelVersion: prov.ModelVersion,
		},
	}
	if _, err := r.coll.InsertOne(ctx, doc); err != nil {
		return err
	}
	detection.ID = id.Hex()
	detection.Status = status
	detection.DetectedAt = detectedAt
	detection.LastUpdated = lastUpdated
	detection.Provenance = prov
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
	if filter.Limit > 0 {
		opts.SetLimit(int64(filter.Limit + 1))
	}
	cur, err := r.coll.Find(ctx, q, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(ctx) }()

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

func (r *DetectionRepository) UpdateStatus(ctx context.Context, id string, from, to domain.DetectionStatus, at time.Time) (*domain.Detection, error) {
	oid, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	if at.IsZero() {
		at = time.Now().UTC()
	} else {
		at = at.UTC()
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var doc detectionDoc
	err = r.coll.FindOneAndUpdate(ctx,
		bson.D{
			{Key: "_id", Value: oid},
			{Key: "status", Value: string(from)},
		},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "status", Value: string(to)},
			{Key: "lastUpdated", Value: at},
		}}},
		opts,
	).Decode(&doc)
	if err != nil {
		return nil, mapFindErr(err)
	}
	detection := doc.toDomain()
	return &detection, nil
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
	if filter.Cursor != nil {
		oid, err := parseObjectID(filter.Cursor.ID)
		if err != nil {
			return nil, err
		}
		q = append(q, bson.E{Key: "$or", Value: bson.A{
			bson.D{{Key: "detectedAt", Value: bson.D{{Key: "$lt", Value: filter.Cursor.DetectedAt}}}},
			bson.D{
				{Key: "detectedAt", Value: filter.Cursor.DetectedAt},
				{Key: "_id", Value: bson.D{{Key: "$lt", Value: oid}}},
			},
		}})
	}
	return q, nil
}

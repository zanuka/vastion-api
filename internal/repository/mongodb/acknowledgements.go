package mongodb

import (
	"context"
	"time"

	"github.com/zanuka/vastion-api/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var _ domain.AcknowledgementRepository = (*AcknowledgementRepository)(nil)

type AcknowledgementRepository struct {
	coll *mongo.Collection
}

func NewAcknowledgementRepository(c *Client) *AcknowledgementRepository {
	return &AcknowledgementRepository{coll: c.db.Collection(collectionAcknowledgements)}
}

type acknowledgementDoc struct {
	ID          bson.ObjectID `bson:"_id"`
	DetectionID bson.ObjectID `bson:"detectionId"`
	Action      string        `bson:"action"`
	Reason      string        `bson:"reason,omitempty"`
	Operator    string        `bson:"operator"`
	FromStatus  string        `bson:"fromStatus"`
	ToStatus    string        `bson:"toStatus"`
	CreatedAt   time.Time     `bson:"createdAt"`
}

func (d acknowledgementDoc) toDomain() domain.Acknowledgement {
	return domain.Acknowledgement{
		ID:          d.ID.Hex(),
		DetectionID: d.DetectionID.Hex(),
		Action:      domain.AckAction(d.Action),
		Reason:      d.Reason,
		Operator:    d.Operator,
		FromStatus:  domain.DetectionStatus(d.FromStatus),
		ToStatus:    domain.DetectionStatus(d.ToStatus),
		CreatedAt:   d.CreatedAt.UTC(),
	}
}

func (r *AcknowledgementRepository) Insert(ctx context.Context, ack *domain.Acknowledgement) error {
	id, err := objectIDOrNew(ack.ID)
	if err != nil {
		return err
	}
	detectionID, err := parseObjectID(ack.DetectionID)
	if err != nil {
		return err
	}
	createdAt := ack.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	doc := acknowledgementDoc{
		ID:          id,
		DetectionID: detectionID,
		Action:      string(ack.Action),
		Reason:      ack.Reason,
		Operator:    ack.Operator,
		FromStatus:  string(ack.FromStatus),
		ToStatus:    string(ack.ToStatus),
		CreatedAt:   createdAt.UTC(),
	}
	if _, err := r.coll.InsertOne(ctx, doc); err != nil {
		return err
	}
	ack.ID = id.Hex()
	ack.CreatedAt = createdAt.UTC()
	return nil
}

func (r *AcknowledgementRepository) GetByID(ctx context.Context, id string) (*domain.Acknowledgement, error) {
	oid, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	var doc acknowledgementDoc
	if err := r.coll.FindOne(ctx, bson.D{{Key: "_id", Value: oid}}).Decode(&doc); err != nil {
		return nil, mapFindErr(err)
	}
	ack := doc.toDomain()
	return &ack, nil
}

func (r *AcknowledgementRepository) ListByDetectionID(ctx context.Context, detectionID string) ([]domain.Acknowledgement, error) {
	oid, err := parseObjectID(detectionID)
	if err != nil {
		return nil, err
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cur, err := r.coll.Find(ctx, bson.D{{Key: "detectionId", Value: oid}}, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(ctx) }()

	var docs []acknowledgementDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]domain.Acknowledgement, 0, len(docs))
	for _, doc := range docs {
		out = append(out, doc.toDomain())
	}
	return out, nil
}

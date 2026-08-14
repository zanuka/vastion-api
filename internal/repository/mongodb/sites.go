package mongodb

import (
	"context"

	"github.com/zanuka/vastion-api/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var _ domain.SiteRepository = (*SiteRepository)(nil)

type SiteRepository struct {
	coll *mongo.Collection
}

func NewSiteRepository(c *Client) *SiteRepository {
	return &SiteRepository{coll: c.db.Collection(collectionSites)}
}

type siteDoc struct {
	ID     bson.ObjectID `bson:"_id"`
	Name   string        `bson:"name"`
	Code   string        `bson:"code"`
	Status string        `bson:"status"`
}

func (d siteDoc) toDomain() domain.Site {
	return domain.Site{
		ID:     d.ID.Hex(),
		Name:   d.Name,
		Code:   d.Code,
		Status: domain.SiteStatus(d.Status),
	}
}

func (r *SiteRepository) Insert(ctx context.Context, site *domain.Site) error {
	id, err := objectIDOrNew(site.ID)
	if err != nil {
		return err
	}
	status := site.Status
	if status == "" {
		status = domain.SiteStatusActive
	}
	doc := siteDoc{
		ID:     id,
		Name:   site.Name,
		Code:   site.Code,
		Status: string(status),
	}
	if _, err := r.coll.InsertOne(ctx, doc); err != nil {
		return err
	}
	site.ID = id.Hex()
	site.Status = status
	return nil
}

func (r *SiteRepository) GetByID(ctx context.Context, id string) (*domain.Site, error) {
	oid, err := parseObjectID(id)
	if err != nil {
		return nil, err
	}
	var doc siteDoc
	if err := r.coll.FindOne(ctx, bson.D{{Key: "_id", Value: oid}}).Decode(&doc); err != nil {
		return nil, mapFindErr(err)
	}
	site := doc.toDomain()
	return &site, nil
}

func (r *SiteRepository) GetByCode(ctx context.Context, code string) (*domain.Site, error) {
	var doc siteDoc
	if err := r.coll.FindOne(ctx, bson.D{{Key: "code", Value: code}}).Decode(&doc); err != nil {
		return nil, mapFindErr(err)
	}
	site := doc.toDomain()
	return &site, nil
}

func (r *SiteRepository) List(ctx context.Context) ([]domain.Site, error) {
	opts := options.Find().SetSort(bson.D{{Key: "code", Value: 1}})
	cur, err := r.coll.Find(ctx, bson.D{}, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(ctx) }()

	var docs []siteDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]domain.Site, 0, len(docs))
	for _, doc := range docs {
		out = append(out, doc.toDomain())
	}
	return out, nil
}

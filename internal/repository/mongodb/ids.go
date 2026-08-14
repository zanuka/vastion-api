package mongodb

import (
	"errors"

	"github.com/zanuka/vastion-api/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func parseObjectID(id string) (bson.ObjectID, error) {
	if id == "" {
		return bson.ObjectID{}, domain.ErrInvalidID
	}
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return bson.ObjectID{}, domain.ErrInvalidID
	}
	return oid, nil
}

func objectIDOrNew(id string) (bson.ObjectID, error) {
	if id == "" {
		return bson.NewObjectID(), nil
	}
	return parseObjectID(id)
}

func mapFindErr(err error) error {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return domain.ErrNotFound
	}
	return err
}

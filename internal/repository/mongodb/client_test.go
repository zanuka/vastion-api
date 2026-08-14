package mongodb

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestDatabaseName(t *testing.T) {
	tests := []struct {
		uri  string
		want string
	}{
		{"mongodb://watchdesk:watchdesk@localhost:27017/watchdesk?authSource=admin", "watchdesk"},
		{"mongodb://localhost:27017/", "watchdesk"},
		{"mongodb://localhost:27017", "watchdesk"},
		{"mongodb://localhost:27017/other", "other"},
	}
	for _, tt := range tests {
		got, err := databaseName(tt.uri)
		if err != nil {
			t.Fatalf("databaseName(%q): %v", tt.uri, err)
		}
		if got != tt.want {
			t.Fatalf("databaseName(%q) = %q, want %q", tt.uri, got, tt.want)
		}
	}
}

func TestParseObjectID(t *testing.T) {
	id := bson.NewObjectID()
	got, err := parseObjectID(id.Hex())
	if err != nil {
		t.Fatal(err)
	}
	if got != id {
		t.Fatalf("got %s want %s", got.Hex(), id.Hex())
	}
	if _, err := parseObjectID(""); err == nil {
		t.Fatal("expected error for empty id")
	}
	if _, err := parseObjectID("not-an-object-id"); err == nil {
		t.Fatal("expected error for invalid id")
	}
}

func TestSiteIndexModels(t *testing.T) {
	models := siteIndexModels()
	if len(models) != 1 {
		t.Fatalf("len = %d, want 1", len(models))
	}
	assertIndexKeys(t, models[0], bson.D{{Key: "code", Value: 1}})
	assertUnique(t, models[0], true)
}

func TestSensorIndexModels(t *testing.T) {
	models := sensorIndexModels()
	if len(models) != 2 {
		t.Fatalf("len = %d, want 2", len(models))
	}
	assertIndexKeys(t, models[0], bson.D{{Key: "siteId", Value: 1}})
	assertUnique(t, models[0], false)
	assertIndexKeys(t, models[1], bson.D{
		{Key: "siteId", Value: 1},
		{Key: "code", Value: 1},
	})
	assertUnique(t, models[1], true)
}

func TestDetectionIndexModels(t *testing.T) {
	models := detectionIndexModels()
	if len(models) != 2 {
		t.Fatalf("len = %d, want 2", len(models))
	}
	assertIndexKeys(t, models[0], bson.D{
		{Key: "siteId", Value: 1},
		{Key: "status", Value: 1},
		{Key: "detectedAt", Value: -1},
	})
	assertIndexKeys(t, models[1], bson.D{{Key: "sensorId", Value: 1}})
}

func TestAcknowledgementIndexModels(t *testing.T) {
	models := acknowledgementIndexModels()
	if len(models) != 1 {
		t.Fatalf("len = %d, want 1", len(models))
	}
	assertIndexKeys(t, models[0], bson.D{
		{Key: "detectionId", Value: 1},
		{Key: "createdAt", Value: -1},
	})
}

func assertIndexKeys(t *testing.T, model mongo.IndexModel, want bson.D) {
	t.Helper()
	got, ok := model.Keys.(bson.D)
	if !ok {
		t.Fatalf("keys type %T, want bson.D", model.Keys)
	}
	if len(got) != len(want) {
		t.Fatalf("keys len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Key != want[i].Key || got[i].Value != want[i].Value {
			t.Fatalf("keys[%d] = {%s %v}, want {%s %v}", i, got[i].Key, got[i].Value, want[i].Key, want[i].Value)
		}
	}
}

func assertUnique(t *testing.T, model mongo.IndexModel, want bool) {
	t.Helper()
	got := false
	if model.Options != nil {
		var opts options.IndexOptions
		for _, fn := range model.Options.List() {
			if err := fn(&opts); err != nil {
				t.Fatal(err)
			}
		}
		got = opts.Unique != nil && *opts.Unique
	}
	if got != want {
		t.Fatalf("unique = %v, want %v", got, want)
	}
}

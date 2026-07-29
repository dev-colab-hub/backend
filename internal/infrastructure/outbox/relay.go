package outbox

import (
	"context"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Publisher forwards an event payload to the outside world (message broker,
// webhook, ...). Implementations must be safe to call repeatedly with the
// same event: the outbox guarantees at-least-once, not exactly-once.
type Publisher interface {
	Publish(ctx context.Context, eventName string, payload []byte) error
}

// SlogPublisher logs events instead of sending them anywhere. Swap it for a
// Kafka/NATS/SQS publisher in a real deployment.
type SlogPublisher struct{}

func (SlogPublisher) Publish(ctx context.Context, eventName string, payload []byte) error {
	slog.InfoContext(ctx, "publishing domain event",
		slog.String("event", eventName), slog.String("payload", string(payload)))
	return nil
}

// relayEventDoc is the projection of an outbox document the relay actually
// needs (no aggregate_id: the relay only forwards name + payload).
type relayEventDoc struct {
	ID          string     `bson:"_id"`
	EventName   string     `bson:"event_name"`
	Payload     []byte     `bson:"payload"`
	PublishedAt *time.Time `bson:"published_at"`
}

// Relay polls the outbox collection and publishes unpublished events. Run
// exactly one instance: there is no findOneAndUpdate-based claim/lock here
// (Mongo's closest equivalent to Postgres's FOR UPDATE SKIP LOCKED), so
// running multiple instances concurrently could publish the same batch
// twice — the same unenforced single-instance assumption as before.
type Relay struct {
	collection *mongo.Collection
	publisher  Publisher
	interval   time.Duration
	batchSize  int64
}

func NewRelay(collection *mongo.Collection, publisher Publisher, interval time.Duration) *Relay {
	return &Relay{
		collection: collection,
		publisher:  publisher,
		interval:   interval,
		batchSize:  100,
	}
}

// Start blocks until ctx is cancelled.
func (r *Relay) Start(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.relayBatch(ctx); err != nil {
				slog.ErrorContext(ctx, "outbox relay batch failed", slog.Any("error", err))
			}
		}
	}
}

func (r *Relay) relayBatch(ctx context.Context) error {
	cursor, err := r.collection.Find(ctx,
		bson.D{{Key: "published_at", Value: nil}},
		options.Find().
			SetSort(bson.D{{Key: "occurred_at", Value: 1}}).
			SetLimit(r.batchSize),
	)
	if err != nil {
		return err
	}

	var events []relayEventDoc
	if err := cursor.All(ctx, &events); err != nil {
		return err
	}

	for _, event := range events {
		if err := r.publisher.Publish(ctx, event.EventName, event.Payload); err != nil {
			// Stop the batch; unpublished events are retried next tick.
			return err
		}

		now := time.Now()
		if _, err := r.collection.UpdateOne(ctx,
			bson.D{{Key: "_id", Value: event.ID}},
			bson.D{{Key: "$set", Value: bson.D{{Key: "published_at", Value: now}}}},
		); err != nil {
			return err
		}
	}

	return nil
}

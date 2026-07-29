package mongodb

import (
	"context"
	"encoding/json"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/sklinkert/go-ddd/internal/domain/events"
)

// insertOutboxEvents stores domain events in the outbox collection. Call it
// with a context bound to the same session/transaction as the aggregate
// write so the state change and its events commit or roll back together.
func insertOutboxEvents(ctx context.Context, coll *mongo.Collection, domainEvents []events.DomainEvent) error {
	for _, event := range domainEvents {
		payload, err := json.Marshal(event)
		if err != nil {
			return err
		}

		doc := outboxEventDoc{
			ID:          event.EventId().String(),
			AggregateID: event.AggregateId().String(),
			EventName:   event.EventName(),
			Payload:     payload,
			OccurredAt:  event.OccurredAt(),
		}

		if _, err := coll.InsertOne(ctx, doc); err != nil {
			return err
		}
	}

	return nil
}

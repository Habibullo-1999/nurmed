package ctxutil

import "context"

type key string

const actorIDKey key = "actor_id"

func WithActorID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, actorIDKey, userID)
}

func ActorID(ctx context.Context) *int64 {
	v, ok := ctx.Value(actorIDKey).(int64)
	if !ok || v == 0 {
		return nil
	}
	return &v
}

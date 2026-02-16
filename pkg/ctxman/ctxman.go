package ctxman

import (
	"context"
	"strconv"
	"time"
)

type CtxKey string

func (c CtxKey) String() string {
	return string(c)
}

const (
	RequestID CtxKey = "requestID"
)

func NewWithRequestID() context.Context {
	// Create a new context with a unique request ID
	return context.WithValue(context.Background(), RequestID, getRequestID())
}

func AddRequestID(ctx context.Context) context.Context {
	return context.WithValue(ctx, RequestID, getRequestID())
}

func UpsertRequestID(ctx context.Context) context.Context {
	if ctx.Value(RequestID) == nil {
		return context.WithValue(ctx, RequestID, getRequestID())
	}
	return ctx
}

func GetRequestID(ctx context.Context) string {
	if reqID, ok := ctx.Value(RequestID).(string); ok {
		return reqID
	}
	return ""
}

func getRequestID() string {
	return "reqid_" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

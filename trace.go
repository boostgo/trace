package trace

import "context"

// TryGet return trace id and state if exists
func TryGet(ctx context.Context, key Key) (string, bool) {
	traceID := ctx.Value(key)
	if traceID == nil {
		return "", false
	}

	traceIdString, ok := traceID.(string)
	if !ok {
		return "", false
	}

	return traceIdString, true
}

// Get calls TryGet but state
func Get(ctx context.Context, key Key) string {
	traceID, _ := TryGet(ctx, key)
	return traceID
}

// Set sets new trace id to provided context
func Set(ctx context.Context, key Key, traceID string) context.Context {
	if _, ok := TryGet(ctx, key); ok {
		return ctx
	}

	return context.WithValue(ctx, key, traceID)
}

// Exist check if any protocol trace id exists
func Exist(ctx context.Context, key Key) bool {
	_, ok := TryGet(ctx, key)
	return ok
}

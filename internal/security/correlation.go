package security

import "context"

type jobIDContextKey struct{}

// WithJobID carries an opaque background job/event identifier through a
// worker operation. It is safe for logs because callers must provide the
// durable ID, never the payload or secret token.
func WithJobID(ctx context.Context, jobID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, jobIDContextKey{}, jobID)
}

func JobIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(jobIDContextKey{}).(string)
	return value
}

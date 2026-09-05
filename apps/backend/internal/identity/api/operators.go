package api

import "context"

// OperatorDirectory is the identity Open Host Service for platform operators.
type OperatorDirectory interface {
	IsPlatformOperator(ctx context.Context, userID string) (bool, error)
}

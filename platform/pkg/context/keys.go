// cmdr: cmdr: context\keys.go

package context

type contextKey string

const (
	// FiberRequestContextKey używany w Fiber (string)
	FiberRequestContextKey = "requestContext"

	// RequestContextKey używany w net/http (dedykowany typ unika SA1029)
	RequestContextKey contextKey = "requestContext"
)

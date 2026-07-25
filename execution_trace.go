package rules

import (
	"context"
	"sync"
)

type traceKey struct{}

// ExecutionTrace records the execution path of each rule reached during tree
// evaluation. It is safe for concurrent use and does not mutate the rules,
// so a single tree (including its rules) can be validated from multiple
// goroutines when a trace is not shared.
type ExecutionTrace struct {
	mu    sync.Mutex
	paths map[Rule]string
}

// WithExecutionTrace returns a context carrying an ExecutionTrace and the
// trace itself. When the returned context is used with Evaluate, Validate,
// or ValidateMulti, the trace records the path of every rule reached by a
// LeafNode.
//
// Example:
//
//	ctx, trace := rules.WithExecutionTrace(ctx)
//	err := rules.ValidateWithData(ctx, tree, hooks, "validate", user)
//	for _, rule := range treeRules {
//	    fmt.Println(rule.Name(), trace.Path(rule))
//	}
func WithExecutionTrace(ctx context.Context) (context.Context, *ExecutionTrace) {
	trace := &ExecutionTrace{paths: make(map[Rule]string)}
	return context.WithValue(ctx, traceKey{}, trace), trace
}

// Path returns the execution path recorded for the given rule, or an empty
// string if the rule was not reached during evaluation.
func (t *ExecutionTrace) Path(rule Rule) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.paths[rule]
}

func (t *ExecutionTrace) record(rule Rule, path string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.paths[rule] = path
}

// traceFromContext returns the ExecutionTrace attached to ctx, or nil.
func traceFromContext(ctx context.Context) *ExecutionTrace {
	trace, _ := ctx.Value(traceKey{}).(*ExecutionTrace)
	return trace
}

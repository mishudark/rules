package rules

import (
	"context"
	"strings"
	"sync"
)

type traceKey struct{}

// ExecutionTrace records the execution path of each rule reached during tree
// evaluation. The trace is carried by the context and used by the node
// traversal: each node pushes its name onto a segment stack while evaluating
// its children and pops it afterwards, and a LeafNode records the joined path
// for every rule it reaches.
//
// A trace is a per-evaluation object: the segment stack is mutated during
// traversal, so the same trace must not be shared across concurrent
// evaluations (mirroring the guidance not to share a context across
// goroutines, §6.3 in AGENTS.md). Reading paths with Path after evaluation
// completes is safe from any goroutine.
type ExecutionTrace struct {
	mu       sync.Mutex
	paths    map[Rule]string
	segments []string
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

// push appends a segment to the current path stack. Called by nodes while
// traversing down into their children.
func (t *ExecutionTrace) push(segment string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.segments = append(t.segments, segment)
}

// pop removes the last segment from the path stack. It must only be called
// after the matching push.
func (t *ExecutionTrace) pop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.segments = t.segments[:len(t.segments)-1]
}

// joinPath returns the current path stack joined with the extra segments,
// e.g. "root -> ageGt30 -> leafNode -> rule1".
func (t *ExecutionTrace) joinPath(extra ...string) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	segs := make([]string, 0, len(t.segments)+len(extra))
	segs = append(segs, t.segments...)
	segs = append(segs, extra...)
	return strings.Join(segs, " -> ")
}

// record stores the execution path for the given rule.
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

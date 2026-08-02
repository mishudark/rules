package rules

import (
	"context"
	"errors"
)

// Hook is called after each step of the validation process.
// It receives the current context and can return an error to halt validation.
type Hook func(ctx context.Context) error

// ProcessingHooks holds the hooks for each step of the validation process.
type ProcessingHooks struct {
	AfterPrepareConditions  Hook
	AfterEvaluateConditions Hook
	AfterPrepareRules       Hook
	AfterValidateRules      Hook
}

// Target holds the Evaluable tree and the context for evaluation.
type Target struct {
	tree Evaluable
	ctx  context.Context
}

// NewTarget creates a new Target.
func NewTarget(ctx context.Context, tree Evaluable) *Target {
	return &Target{
		tree: tree,
		ctx:  ctx,
	}
}

// run executes the four-phase engine pipeline over a batch of targets:
//
//  1. PrepareConditions for every target (fan out / batch fetches)
//  2. Evaluate every target and collect candidate rules
//  3. Prepare every candidate rule (batch)
//  4. Validate every prepared rule
//
// It is shared by Validate, ValidateMulti, EvaluateMetrics and
// EvaluateMetricsMulti. When collectMetrics is true, an outcome collector is
// attached per target during phase 4 and one Report per target is returned.
//
// targets is copied before per-target state is attached, so the caller's
// slice is never mutated. Hooks receive the ctx passed by the caller: per-
// target prepared data is not visible at the hook level, which is consistent
// across single- and multi-target runs.
func run(ctx context.Context, targets []Target, hooks ProcessingHooks, name string, collectMetrics bool) ([]Report, []error) {
	// Copy so the caller's slice is never mutated: each target gets its own
	// prepared-data store, so prepare results for one target never leak into
	// another when a tree is shared between targets.
	targets = append([]Target(nil), targets...)
	for i := range targets {
		targets[i].ctx, _ = withPreparedStore(targets[i].ctx)
	}

	// Phase 1: prepare the conditions for all targets.
	for _, target := range targets {
		if err := target.tree.PrepareConditions(target.ctx); err != nil {
			return nil, []error{err}
		}
	}

	if hooks.AfterPrepareConditions != nil {
		if err := hooks.AfterPrepareConditions(ctx); err != nil {
			return nil, []error{err}
		}
	}

	// Phase 2: evaluate all targets and collect candidate rules. The name is
	// pushed onto the execution trace (if any) as the root path segment.
	evaluated := make([][]Rule, len(targets))
	for i, target := range targets {
		if trace := traceFromContext(target.ctx); trace != nil {
			trace.push(name)
		}
		_, evaluated[i] = target.tree.Evaluate(target.ctx)
		if trace := traceFromContext(target.ctx); trace != nil {
			trace.pop()
		}
	}

	if hooks.AfterEvaluateConditions != nil {
		if err := hooks.AfterEvaluateConditions(ctx); err != nil {
			return nil, []error{err}
		}
	}

	// Phase 3: prepare all rules across targets (batch).
	targetErrs := make([][]error, len(targets))
	prepared := make([][]Rule, len(targets))
	for i, target := range targets {
		for _, rule := range evaluated[i] {
			if _, err := rule.Prepare(target.ctx); err != nil {
				targetErrs[i] = append(targetErrs[i], err)
				continue
			}
			prepared[i] = append(prepared[i], rule)
		}
	}

	if hooks.AfterPrepareRules != nil {
		if err := hooks.AfterPrepareRules(ctx); err != nil {
			return nil, []error{err}
		}
	}

	// Phase 4: validate prepared rules per target, optionally collecting
	// metric outcomes. The validation context carries an outcome collector
	// so metric-carrying rules can record their outcomes while they
	// validate; the collector is a per-evaluation side channel, so no rule
	// is mutated and rules stay safe to share across goroutines.
	reports := make([]Report, len(targets))
	for i, target := range targets {
		valCtx := target.ctx
		var collector *outcomeCollector
		if collectMetrics {
			valCtx, collector = withOutcomeCollector(target.ctx)
		}

		for _, rule := range prepared[i] {
			if err := rule.Validate(valCtx); err != nil {
				targetErrs[i] = append(targetErrs[i], err)
			}
		}

		if collector != nil {
			// Surface any errors carried by emitted outcomes.
			for _, outcome := range collector.outcomes {
				if outcome.Err != nil {
					targetErrs[i] = append(targetErrs[i], outcome.Err)
				}
			}

			reports[i] = aggregateOutcomes(collector.outcomes)
			reports[i].Errors = targetErrs[i]
			reports[i].Valid = len(targetErrs[i]) == 0
		}
	}

	if hooks.AfterValidateRules != nil {
		if err := hooks.AfterValidateRules(ctx); err != nil {
			for i := range reports {
				reports[i].Errors = append(reports[i].Errors, err)
				reports[i].Valid = false
			}
			return reports, append(flattenErrors(targetErrs), err)
		}
	}

	return reports, flattenErrors(targetErrs)
}

// ValidateMulti executes the targets trees in 4 steps:
// 1. Prepare the conditions for evaluation
// 2. Evaluate the tree and get candidate rules
// 3. Prepare the rule for evaluation
// 4. Validate the prepared rules
//
// targets is not mutated: per-target state is attached to a copy. name is
// used as the root label when execution tracing is enabled.
func ValidateMulti(ctx context.Context, targets []Target, hooks ProcessingHooks, name string) error {
	_, errs := run(ctx, targets, hooks, name, false)
	return joinErrors(errs)
}

// Validate executes the Evaluable tree in 4 steps:
// 1. Prepare the conditions for evaluation
// 2. Evaluate the tree and get candidate rules
// 3. Prepare the rule for evaluation
// 4. Validate the prepared rules
//
// name is used as the root label when execution tracing is enabled.
func Validate(ctx context.Context, tree Evaluable, hooks ProcessingHooks, name string) error {
	_, errs := run(ctx, []Target{{tree: tree, ctx: ctx}}, hooks, name, false)
	return joinErrors(errs)
}

// EvaluateMetrics executes the tree in the same 4 steps as Validate, but also
// collects and aggregates the metric indicators reached during evaluation.
// Validation rules keep their error semantics; metric outcomes are
// aggregated into a Report keyed by metric name.
//
// The two-phase dataloader invariant is preserved: every condition is
// prepared before any rule is prepared, and every rule is prepared before any
// validation runs.
//
// Metrics are only collected when this entry point (or EvaluateMetricsMulti)
// is used; calling Validate on a tree that contains metric-carrying rules
// simply ignores their outcomes.
func EvaluateMetrics(ctx context.Context, tree Evaluable, hooks ProcessingHooks, name string) (Report, error) {
	reports, errs := run(ctx, []Target{{tree: tree, ctx: ctx}}, hooks, name, true)
	err := joinErrors(errs)
	if len(reports) == 0 {
		return Report{}, err
	}
	return reports[0], err
}

// EvaluateMetricsMulti executes the targets' trees in the same 4 steps as
// ValidateMulti, collecting and aggregating the metric outcomes carried by
// metric-carrying rules per target. It returns one Report per target, in the
// same order as the targets slice.
//
// The dataloader batching invariant extends across targets: all conditions
// are prepared before any evaluation, and all rules are prepared before any
// validation runs.
func EvaluateMetricsMulti(ctx context.Context, targets []Target, hooks ProcessingHooks, name string) ([]Report, error) {
	reports, errs := run(ctx, targets, hooks, name, true)
	return reports, joinErrors(errs)
}

// flattenErrors flattens a slice of per-target error slices into a single
// error slice.
func flattenErrors(targetErrs [][]error) []error {
	var errs []error
	for _, target := range targetErrs {
		errs = append(errs, target...)
	}
	return errs
}

// joinErrors joins the given errors, returning nil when there are none. A
// single error is returned bare so its identity is preserved for callers
// that compare errors directly.
func joinErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}
	return errors.Join(errs...)
}

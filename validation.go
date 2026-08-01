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

// ValidateMulti executes the targets trees in 4 steps:
// 1. Prepare the conditions for evaluation
// 2. Evaluate the tree and get candidate rules
// 3. Prepare the rule for evaluation
// 4. Validate the prepared rules
func ValidateMulti(ctx context.Context, targets []Target, hooks ProcessingHooks, name string) error {
	for _, target := range targets {
		// Prepare the conditions for evaluation
		err := target.tree.PrepareConditions(target.ctx)
		if err != nil {
			// If there is an error, return immediately
			return err
		}
	}

	if hooks.AfterPrepareConditions != nil {
		if err := hooks.AfterPrepareConditions(ctx); err != nil {
			return err
		}
	}

	// evaluatedRules is a map of target index to rules from evaluation
	evaluatedRules := make(map[int][]Rule)

	for i, target := range targets {
		// Evaluate the tree and get candidate rules
		_, rules := target.tree.Evaluate(target.ctx, name)
		evaluatedRules[i] = rules
	}

	if hooks.AfterEvaluateConditions != nil {
		if err := hooks.AfterEvaluateConditions(ctx); err != nil {
			return err
		}
	}

	// rules is a map of target index to prepared rules
	preparedRules := make(map[int][]Rule)
	// create a slice to hold errors
	errs := make([]error, 0, len(targets))

	for i := range targets {
		// prepare the rule for evaluation
		rules := evaluatedRules[i]
		preparedRules[i] = make([]Rule, 0, len(rules))

		for _, rule := range rules {
			err := rule.Prepare(targets[i].ctx)
			if err != nil {
				// If the rule is not valid, append the error and continue
				errs = append(errs, err)
				continue
			}

			// If the rule is valid, append it to the prepared rules
			preparedRules[i] = append(preparedRules[i], rule)
		}
	}

	if hooks.AfterPrepareRules != nil {
		if err := hooks.AfterPrepareRules(ctx); err != nil {
			return err
		}
	}

	// Prepare errors are preserved; validate errors are appended below

	for i := range targets {
		// Validate prepared rules
		for _, rule := range preparedRules[i] {
			err := rule.Validate(targets[i].ctx)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	if hooks.AfterValidateRules != nil {
		if err := hooks.AfterValidateRules(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// Validate executes the Evaluable tree in 4 steps:
// 1. Prepare the conditions for evaluation
// 2. Evaluate the tree and get candidate rules
// 3. Prepare the rule for evaluation
// 4. Validate the prepared rules
func Validate(ctx context.Context, tree Evaluable, hooks ProcessingHooks, name string) error {
	// Prepare the conditions for evaluation
	err := tree.PrepareConditions(ctx)
	if err != nil {
		return err
	}

	if hooks.AfterPrepareConditions != nil {
		if err := hooks.AfterPrepareConditions(ctx); err != nil {
			return err
		}
	}

	// Evaluate the tree and get candidate rules
	_, rules := tree.Evaluate(ctx, name)

	if hooks.AfterEvaluateConditions != nil {
		if err := hooks.AfterEvaluateConditions(ctx); err != nil {
			return err
		}
	}

	// create slices to hold errors and prepared rules
	errs := make([]error, 0, len(rules))
	preparedRules := make([]Rule, 0, len(rules))

	// Prepare the rule for evaluation
	for _, rule := range rules {
		err := rule.Prepare(ctx)
		if err != nil {
			// If the rule is not valid, append the error and continue
			errs = append(errs, err)
			continue
		}

		// If the rule is valid, append it to the prepared rules
		preparedRules = append(preparedRules, rule)
	}

	if hooks.AfterPrepareRules != nil {
		if err := hooks.AfterPrepareRules(ctx); err != nil {
			return err
		}
	}

	// Validate prepared rules
	for _, rule := range preparedRules {
		err := rule.Validate(ctx)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if hooks.AfterValidateRules != nil {
		if err := hooks.AfterValidateRules(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// EvaluateMetrics executes the tree in the same 4 steps as Validate, but also
// collects and aggregates the Metric indicators reached during evaluation.
// EvaluateMetrics executes the tree in the same 4 steps as Validate, but also
// collects and aggregates the metric outcomes carried by metric-carrying
// rules. Validation rules keep their error semantics; metric outcomes are
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
	// Prepare the conditions for evaluation
	err := tree.PrepareConditions(ctx)
	if err != nil {
		return Report{}, err
	}

	if hooks.AfterPrepareConditions != nil {
		if err := hooks.AfterPrepareConditions(ctx); err != nil {
			return Report{}, err
		}
	}

	// Evaluate the tree and get candidate rules
	_, rules := tree.Evaluate(ctx, name)

	if hooks.AfterEvaluateConditions != nil {
		if err := hooks.AfterEvaluateConditions(ctx); err != nil {
			return Report{}, err
		}
	}

	errs := make([]error, 0, len(rules))
	preparedRules := make([]Rule, 0, len(rules))

	// Prepare the rule for evaluation
	for _, rule := range rules {
		err := rule.Prepare(ctx)
		if err != nil {
			// If the rule is not valid, append the error and continue
			errs = append(errs, err)
			continue
		}

		// If the rule is valid, append it to the prepared rules
		preparedRules = append(preparedRules, rule)
	}

	if hooks.AfterPrepareRules != nil {
		if err := hooks.AfterPrepareRules(ctx); err != nil {
			return Report{}, err
		}
	}

	// Validate prepared rules. The validation context carries an outcome
	// collector so metric-carrying rules can record their outcomes while
	// they validate; the collector is a per-evaluation side channel, so no
	// rule is mutated and pure rules stay safe to share across goroutines.
	valCtx, collector := withOutcomeCollector(ctx)
	for _, rule := range preparedRules {
		err := rule.Validate(valCtx)
		if err != nil {
			errs = append(errs, err)
		}
	}

	// Surface any errors carried by emitted outcomes
	for _, outcome := range collector.outcomes {
		if outcome.Err != nil {
			errs = append(errs, outcome.Err)
		}
	}

	if hooks.AfterValidateRules != nil {
		if err := hooks.AfterValidateRules(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	report := aggregateOutcomes(collector.outcomes)
	report.Errors = errs
	report.Valid = len(errs) == 0
	return report, errors.Join(errs...)
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
	// Phase A: prepare the conditions for all targets
	for _, target := range targets {
		err := target.tree.PrepareConditions(target.ctx)
		if err != nil {
			return nil, err
		}
	}

	if hooks.AfterPrepareConditions != nil {
		if err := hooks.AfterPrepareConditions(ctx); err != nil {
			return nil, err
		}
	}

	// Phase B: evaluate all targets and collect candidate rules
	results := make([][]Rule, len(targets))

	for i, target := range targets {
		_, rules := target.tree.Evaluate(target.ctx, name)
		results[i] = rules
	}

	if hooks.AfterEvaluateConditions != nil {
		if err := hooks.AfterEvaluateConditions(ctx); err != nil {
			return nil, err
		}
	}

	// Phase C: prepare all rules across targets (batch)
	targetErrs := make([][]error, len(targets))
	preparedRules := make([][]Rule, len(targets))

	for i, target := range targets {
		for _, rule := range results[i] {
			if err := rule.Prepare(target.ctx); err != nil {
				targetErrs[i] = append(targetErrs[i], err)
				continue
			}
			preparedRules[i] = append(preparedRules[i], rule)
		}
	}

	if hooks.AfterPrepareRules != nil {
		if err := hooks.AfterPrepareRules(ctx); err != nil {
			return nil, err
		}
	}

	// Phase D: validate rules per target, collecting their metric outcomes
	reports := make([]Report, len(targets))
	for i, target := range targets {
		valCtx, collector := withOutcomeCollector(target.ctx)
		for _, rule := range preparedRules[i] {
			if err := rule.Validate(valCtx); err != nil {
				targetErrs[i] = append(targetErrs[i], err)
			}
		}

		for _, outcome := range collector.outcomes {
			if outcome.Err != nil {
				targetErrs[i] = append(targetErrs[i], outcome.Err)
			}
		}

		reports[i] = aggregateOutcomes(collector.outcomes)
		reports[i].Errors = targetErrs[i]
		reports[i].Valid = len(targetErrs[i]) == 0
	}

	if hooks.AfterValidateRules != nil {
		if err := hooks.AfterValidateRules(ctx); err != nil {
			for i := range reports {
				reports[i].Errors = append(reports[i].Errors, err)
				reports[i].Valid = false
			}
			return reports, err
		}
	}

	return reports, errors.Join(flattenErrors(targetErrs)...)
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

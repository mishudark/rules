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
	// rulesCounter is used to count the number of rules
	rulesCounter := 0

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
			rulesCounter++
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
			return err
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
			return err
		}
	}

	return errors.Join(errs...)
}

package rules

import (
	"context"
)

// Validate executes the Evaluable tree in 4 steps:
// 1. Prepare the conditions for evaluation
// 2. Evaluate the tree and get candidate rules
// 3. Prepare the rule for evaluation
// 4. Validate the prepared rules
func Validate(ctx context.Context, tree Evaluable, name string) []error {

	// Prepare the conditions for evaluation
	err := tree.PrepareConditions(ctx)
	if err != nil {
		return []error{err}
	}

	// Evaluate the tree and get candidate rules
	_, rules := tree.Evaluate(ctx, name)

	// create slices to hold errors and prepared rules
	errs := make([]error, 0, len(rules))
	preparedRules := make([]Rule, 0, len(rules))

	// Prepare the rule for evaluation
	for _, rule := range rules {
		if err := rule.Prepare(ctx); err != nil {
			// If the rule is not valid, append the error and continue
			errs = append(errs, err)
			continue
		}

		// If the rule is valid, append it to the prepared rules
		preparedRules = append(preparedRules, rule)
	}

	// Validate prepared rules
	for _, rule := range preparedRules {
		if err := rule.Validate(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

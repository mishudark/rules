package validators

import (
	"fmt"
	"math"

	"github.com/mishudark/rules"
)

// number is a constraint that permits any integer or floating-point type.
type number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// stepValueValidator validates that a value is a multiple of a given step.
// An offset can be provided to shift the validation.
// It uses a small tolerance for floating-point comparisons.
func stepValueValidator[T number](value, step, offset T) error {
	if step == 0 {
		return fmt.Errorf("step cannot be zero")
	}

	valFloat := float64(value)
	stepFloat := float64(step)
	offsetFloat := float64(offset)

	remainder := math.Mod(valFloat-offsetFloat, stepFloat)

	// Use a small tolerance for floating point comparisons
	tolerance := 1e-9
	if math.Abs(remainder) > tolerance && math.Abs(remainder-stepFloat) > tolerance {
		return fmt.Errorf("value %v is not a multiple of step %v (with offset %v)", value, step, offset)
	}

	return nil
}

// NewStepValue returns a new Rule that validates if a number is a multiple of a given step.
func StepValue[T number](value T, step, offset T) rules.Rule {
	return rules.NewRulePure("step_value_validator", func() error {
		return stepValueValidator(value, step, offset)
	})
}

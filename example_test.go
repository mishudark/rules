package rules_test

import (
	"context"
	"fmt"

	"github.com/mishudark/rules"
)

type exampleUser struct {
	Name string
	Age  int
}

// Validate a one-shot tree built with closures. The data (the user) is bound
// at construction time, so this style suits single-use validations.
func ExampleValidate() {
	ada := exampleUser{Name: "Ada", Age: 17}

	tree := rules.Root(
		rules.Node(
			rules.NewConditionPure("isAdult", func() bool { return ada.Age >= 18 }),
			rules.Rules(
				rules.NewRulePure("welcome", func() error { return nil }),
			),
		),
	)

	err := rules.Validate(context.Background(), tree, rules.ProcessingHooks{}, "signup")
	fmt.Println(err)

	// Output:
	// <nil>
}

// The data-registry pattern: build the tree once, bind the data at validation
// time. The same tree can be reused with different data, even concurrently.
func ExampleValidateWithData() {
	tree := rules.Rules(
		rules.NewTypedRule("adult", func(ctx context.Context, u exampleUser) error {
			if u.Age < 18 {
				return rules.Error{Field: "age", Err: "must be at least 18", Code: "MIN_AGE"}
			}
			return nil
		}),
	)

	err := rules.ValidateWithData(context.Background(), tree, rules.ProcessingHooks{}, "signup", exampleUser{Name: "Ada", Age: 17})
	fmt.Println(err)

	err = rules.ValidateWithData(context.Background(), tree, rules.ProcessingHooks{}, "signup", exampleUser{Name: "Grace", Age: 36})
	fmt.Println(err)

	// Output:
	// code: MIN_AGE, field: age, error: must be at least 18
	// <nil>
}

// Either picks a branch based on a condition; Not negates one.
func ExampleEither() {
	tree := rules.Either(
		rules.NewCondition("isAdult", func(ctx context.Context) bool {
			u, ok := rules.GetAs[exampleUser](ctx)
			return ok && u.Age >= 18
		}),
		[]rules.Evaluable{
			rules.Rules(rules.NewTypedRule("adultRule", func(ctx context.Context, u exampleUser) error {
				return nil
			})),
		},
		[]rules.Evaluable{
			rules.Rules(rules.NewTypedRule("minorRule", func(ctx context.Context, u exampleUser) error {
				return rules.Error{Field: "age", Err: "minors need a guardian", Code: "GUARDIAN_REQUIRED"}
			})),
		},
	)

	err := rules.ValidateWithData(context.Background(), tree, rules.ProcessingHooks{}, "booking", exampleUser{Name: "Ada", Age: 17})
	fmt.Println(err)

	// Output:
	// code: GUARDIAN_REQUIRED, field: age, error: minors need a guardian
}

// EvaluateMetrics runs the same four phases as Validate and additionally
// aggregates the outcomes emitted by metric-carrying rules, keyed by name.
func ExampleEvaluateMetricsWithData() {
	tree := rules.Rules(
		rules.NewTypedMetricRule("balance", rules.KindCounter, "balance",
			func(ctx context.Context, u exampleUser) (rules.Outcome, error) {
				return rules.CounterValue(42), nil
			}),
	)

	report, err := rules.EvaluateMetricsWithData(context.Background(), tree, rules.ProcessingHooks{}, "health", exampleUser{Name: "Ada", Age: 36})
	fmt.Println("valid:", report.Valid, "err:", err)
	fmt.Println("balance:", report.Metrics["balance"].Count)

	// Output:
	// valid: true err: <nil>
	// balance: 42
}

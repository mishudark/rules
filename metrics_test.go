package rules

import (
	"context"
	"fmt"
	"math"
	"sync"
	"testing"
)

type metricUser struct {
	Name    string
	Balance float64
}

// loggingMetricRule is a metric-carrying Rule that records prepare/validate
// events in order, mirroring the architecture_test event log pattern.
type loggingMetricRule struct {
	name string
	log  *eventLog
}

func (r *loggingMetricRule) Name() string                 { return r.name }
func (r *loggingMetricRule) SetExecutionPath(path string) {}
func (r *loggingMetricRule) GetExecutionPath() string     { return "" }
func (r *loggingMetricRule) Prepare(context.Context) (any, error) {
	r.log.add("prepareRule:%s", r.name)
	return nil, nil
}
func (r *loggingMetricRule) Validate(ctx context.Context) error {
	r.log.add("validateRule:%s", r.name)
	o := CounterValue(1)
	o.Name = r.name
	Emit(ctx, o)
	return nil
}

var _ Rule = (*loggingMetricRule)(nil)

func TestHistogram_Observe(t *testing.T) {
	t.Parallel()

	hist := NewHistogram([]float64{10, 20, 30, math.Inf(1)})
	hist.Observe(5)
	hist.Observe(15)
	hist.Observe(25)
	hist.Observe(50)

	if hist.Total != 4 {
		t.Errorf("Total = %d, want 4", hist.Total)
	}
	if hist.Sum != 95 {
		t.Errorf("Sum = %v, want 95", hist.Sum)
	}
	want := []uint64{1, 2, 3, 4}
	for i, c := range hist.Counts {
		if c != want[i] {
			t.Errorf("Counts[%d] = %d, want %d (le %v)", i, c, want[i], hist.Buckets[i])
		}
	}
}

func TestEvaluateMetrics_Counter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		tree    Evaluable
		wantKey string
		wantVal float64
	}{
		{
			name: "single counter",
			tree: Rules(NewMetricRulePure("mrr", KindCounter, "mrr", func() (Outcome, error) {
				return CounterValue(1250.5), nil
			})),
			wantKey: "mrr",
			wantVal: 1250.5,
		},
		{
			name: "same-name counters sum",
			tree: Rules(
				NewMetricRulePure("revenue", KindCounter, "revenue", func() (Outcome, error) {
					return CounterValue(100), nil
				}),
				NewMetricRulePure("revenue", KindCounter, "revenue", func() (Outcome, error) {
					return CounterValue(250), nil
				}),
			),
			wantKey: "revenue",
			wantVal: 350,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			report, err := EvaluateMetricsWithData(context.Background(), tc.tree, ProcessingHooks{}, "check", "data")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !report.Valid {
				t.Errorf("Valid = false, want true; errors: %v", report.Errors)
			}
			outcome, ok := report.Metrics[tc.wantKey]
			if !ok {
				t.Fatalf("metric %q not found in %v", tc.wantKey, report.Metrics)
			}
			if outcome.Kind != KindCounter {
				t.Errorf("Kind = %v, want counter", outcome.Kind)
			}
			if outcome.Count != tc.wantVal {
				t.Errorf("Count = %v, want %v", outcome.Count, tc.wantVal)
			}
		})
	}
}

func TestEvaluateMetrics_CustomRuleEmitsMetrics(t *testing.T) {
	t.Parallel()

	// The "extend a rule" path: any rule can carry metrics by calling Emit.
	// Two same-name outcomes with AggMax combine to the largest value.
	maxEmit := func(name string, value float64) Rule {
		return NewTypedRule[string](name, func(ctx context.Context, _ string) error {
			o := CounterValue(value)
			o.Name = name
			o.Aggregation = AggMax
			Emit(ctx, o)
			return nil
		})
	}

	tree := Rules(
		maxEmit("peak", 10),
		maxEmit("peak", 42),
	)

	report, err := EvaluateMetricsWithData(context.Background(), tree, ProcessingHooks{}, "check", "data")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Metrics["peak"].Count != 42 {
		t.Errorf("peak = %v, want 42", report.Metrics["peak"].Count)
	}
}

func TestEvaluateMetrics_Histogram(t *testing.T) {
	t.Parallel()

	buckets := []float64{10, 20, 30, math.Inf(1)}

	build := func(values ...float64) Rule {
		return NewMetricRulePure("latency", KindHistogram, "latency_ms", func() (Outcome, error) {
			hist := NewHistogram(buckets)
			for _, v := range values {
				hist.Observe(v)
			}
			return HistogramValue(hist), nil
		})
	}

	tree := Rules(build(5, 15, 25), build(50, 100))

	report, err := EvaluateMetricsWithData(context.Background(), tree, ProcessingHooks{}, "check", "data")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hist := report.Metrics["latency"].Histogram
	want := []uint64{1, 2, 3, 5}
	for i, c := range hist.Counts {
		if c != want[i] {
			t.Errorf("Counts[%d] = %d, want %d", i, c, want[i])
		}
	}
	if hist.Total != 5 {
		t.Errorf("Total = %d, want 5", hist.Total)
	}
	if hist.Sum != 195 {
		t.Errorf("Sum = %v, want 195", hist.Sum)
	}
}

func TestEvaluateMetrics_Score(t *testing.T) {
	t.Parallel()

	tree := Rules(
		NewMetricRulePure("risk", KindScore, "risk", func() (Outcome, error) {
			return ScoreValue(60, 2), nil
		}),
		NewMetricRulePure("risk", KindScore, "risk", func() (Outcome, error) {
			return ScoreValue(90, 1), nil
		}),
	)

	report, err := EvaluateMetricsWithData(context.Background(), tree, ProcessingHooks{}, "check", "data")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Metrics["risk"].Score != 70 {
		t.Errorf("Score = %v, want 70", report.Metrics["risk"].Score)
	}
}

func TestEvaluateMetrics_Valid(t *testing.T) {
	t.Parallel()

	t.Run("reports boolean observation", func(t *testing.T) {
		t.Parallel()

		tree := Rules(NewMetricRulePure("compliant", KindValid, "compliant", func() (Outcome, error) {
			return ValidValue(false, nil), nil // observation only, rule passes
		}))

		report, err := EvaluateMetricsWithData(context.Background(), tree, ProcessingHooks{}, "check", "data")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !report.Valid {
			t.Errorf("Valid = false, want true (rule passed)")
		}
		if report.Metrics["compliant"].Valid {
			t.Errorf("metric Valid = true, want false")
		}
	})

	t.Run("outcome error surfaces in report", func(t *testing.T) {
		t.Parallel()

		tree := Rules(NewMetricRulePure("compliant", KindValid, "compliant", func() (Outcome, error) {
			return ValidValue(false, fmt.Errorf("not compliant")), nil
		}))

		report, err := EvaluateMetricsWithData(context.Background(), tree, ProcessingHooks{}, "check", "data")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if report.Valid {
			t.Errorf("Valid = true, want false")
		}
	})
}

func TestEvaluateMetrics_RuleErrorAndMetricTogether(t *testing.T) {
	t.Parallel()

	// A metric-carrying rule that also fails validation: the failure is
	// reported and the observation is still aggregated (e.g. count attempts).
	tree := Rules(NewMetricRulePure("attempts", KindCounter, "attempts", func() (Outcome, error) {
		return CounterValue(1), Error{Field: "attempts", Err: "failed", Code: "FAILED"}
	}))

	report, err := EvaluateMetricsWithData(context.Background(), tree, ProcessingHooks{}, "check", "data")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if report.Valid {
		t.Errorf("Valid = true, want false")
	}
	if report.Metrics["attempts"].Count != 1 {
		t.Errorf("attempts = %v, want 1", report.Metrics["attempts"].Count)
	}
}

func TestEvaluateMetrics_GatedByCondition(t *testing.T) {
	t.Parallel()

	tree := Node(
		NewConditionPure("isPremium", func() bool { return false }),
		Rules(NewMetricRulePure("mrr", KindCounter, "mrr", func() (Outcome, error) {
			return CounterValue(100), nil
		})),
	)

	report, err := EvaluateMetricsWithData(context.Background(), tree, ProcessingHooks{}, "check", "data")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.Metrics) != 0 {
		t.Errorf("Metrics = %v, want empty (condition was false)", report.Metrics)
	}
	if !report.Valid {
		t.Errorf("Valid = false, want true")
	}
}

func TestEvaluateMetrics_Either(t *testing.T) {
	t.Parallel()

	tree := Either(
		NewConditionPure("isPremium", func() bool { return true }),
		[]Evaluable{
			Rules(NewMetricRulePure("premiumMetric", KindCounter, "premium", func() (Outcome, error) {
				return CounterValue(1), nil
			})),
		},
		[]Evaluable{
			Rules(NewMetricRulePure("freeMetric", KindCounter, "free", func() (Outcome, error) {
				return CounterValue(2), nil
			})),
		},
	)

	report, err := EvaluateMetricsWithData(context.Background(), tree, ProcessingHooks{}, "check", "data")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := report.Metrics["premiumMetric"]; !ok {
		t.Errorf("premiumMetric missing from %v", report.Metrics)
	}
	if _, ok := report.Metrics["freeMetric"]; ok {
		t.Errorf("freeMetric should not be emitted (condition was true)")
	}
}

func TestEvaluateMetrics_MixedTree(t *testing.T) {
	t.Parallel()

	failRule := NewTypedRule[metricUser]("mustFail", func(ctx context.Context, _ metricUser) error {
		return Error{Field: "x", Err: "boom", Code: "BOOM"}
	})

	tree := Root(
		Rules(failRule),
		Rules(
			NewTypedMetricRule[metricUser]("balance", KindCounter, "balance", func(ctx context.Context, u metricUser) (Outcome, error) {
				return CounterValue(u.Balance), nil
			}),
			NewTypedMetricRule[metricUser]("score", KindScore, "score", func(ctx context.Context, u metricUser) (Outcome, error) {
				return ScoreValue(u.Balance/10, 1), nil
			}),
		),
	)

	report, err := EvaluateMetricsWithData(context.Background(), tree, ProcessingHooks{}, "check", metricUser{Balance: 500})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if report.Valid {
		t.Errorf("Valid = true, want false")
	}
	if len(report.Errors) != 1 {
		t.Fatalf("len(Errors) = %d, want 1", len(report.Errors))
	}
	if report.Metrics["balance"].Count != 500 {
		t.Errorf("balance = %v, want 500", report.Metrics["balance"].Count)
	}
	if report.Metrics["score"].Score != 50 {
		t.Errorf("score = %v, want 50", report.Metrics["score"].Score)
	}
}

func TestEvaluateMetrics_MetricRuleWithPrepare(t *testing.T) {
	t.Parallel()

	prepared := false

	rule := NewTypedMetricRuleWithPrepare[metricUser, float64](
		"engagement", KindScore, "engagement",
		func(ctx context.Context, u metricUser) (float64, error) {
			prepared = true
			return u.Balance * 2, nil
		},
		func(ctx context.Context, u metricUser, loaded float64) (Outcome, error) {
			if !prepared {
				t.Error("Validate called before Prepare")
			}
			return ScoreValue(loaded, 1), nil
		},
	)

	report, err := EvaluateMetricsWithData(context.Background(), Rules(rule), ProcessingHooks{}, "check", metricUser{Balance: 40})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !prepared {
		t.Fatal("Prepare was not called")
	}
	if report.Metrics["engagement"].Score != 80 {
		t.Errorf("Score = %v, want 80", report.Metrics["engagement"].Score)
	}
}

func TestEvaluateMetrics_PhaseOrdering(t *testing.T) {
	t.Parallel()

	log := &eventLog{}

	cond := &loggingCondition{name: "cond", log: log, valid: true}
	rule := &loggingRule{name: "rule", log: log}
	metricRule := &loggingMetricRule{name: "metricRule", log: log}

	tree := Node(cond, Rules(rule, metricRule))

	report, err := EvaluateMetrics(context.Background(), tree, ProcessingHooks{}, "check")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The existing architecture invariant applies to metric-carrying rules
	// too: every condition Prepare before any rule Prepare, and every rule
	// Prepare before any Validate.
	if i, j := log.lastIndexOf("prepareCondition:"), log.indexOf("prepareRule:"); i != -1 && i > j {
		t.Errorf("condition prepared after rule prepare; events: %v", log.events)
	}
	if i, j := log.lastIndexOf("prepareRule:"), log.indexOf("validateRule:"); i > j {
		t.Errorf("rule prepared after validation; events: %v", log.events)
	}

	// Metric-carrying rules emit their outcome during Validate.
	if _, ok := report.Metrics["metricRule"]; !ok {
		t.Errorf("metricRule outcome missing from %v", report.Metrics)
	}
}

func TestEvaluateMetricsMulti(t *testing.T) {
	t.Parallel()

	tree := Rules(
		NewTypedMetricRule[metricUser]("balance", KindCounter, "balance", func(ctx context.Context, u metricUser) (Outcome, error) {
			return CounterValue(u.Balance), nil
		}),
	)

	targets := []TreeAndData{
		{Tree: tree, Data: metricUser{Name: "alice", Balance: 10}},
		{Tree: tree, Data: metricUser{Name: "bob", Balance: 20}},
	}

	reports, err := EvaluateMetricsMultiWithData(context.Background(), targets, ProcessingHooks{}, "check")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reports) != 2 {
		t.Fatalf("len(reports) = %d, want 2", len(reports))
	}
	if reports[0].Metrics["balance"].Count != 10 {
		t.Errorf("alice balance = %v, want 10", reports[0].Metrics["balance"].Count)
	}
	if reports[1].Metrics["balance"].Count != 20 {
		t.Errorf("bob balance = %v, want 20", reports[1].Metrics["balance"].Count)
	}
	if !reports[0].Valid || !reports[1].Valid {
		t.Errorf("reports not valid: %v, %v", reports[0].Errors, reports[1].Errors)
	}
}

func TestValidate_IgnoresMetrics(t *testing.T) {
	t.Parallel()

	// Backward compatibility: a tree whose rules carry metric outcomes must
	// behave exactly like a validation-only tree under the plain Validate
	// entry point (no collector in context, so Emit is a no-op).
	tree := Rules(NewMetricRulePure("mrr", KindCounter, "mrr", func() (Outcome, error) {
		return CounterValue(10), nil
	}))

	err := ValidateWithData(context.Background(), tree, ProcessingHooks{}, "check", "data")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEvaluateMetrics_Concurrent(t *testing.T) {
	t.Parallel()

	// Pure metric-carrying rules are safe to share across goroutines: the
	// outcome collector is a per-evaluation side channel and rules are never
	// mutated.
	tree := Rules(
		NewTypedMetricRule[metricUser]("balance", KindCounter, "balance", func(ctx context.Context, u metricUser) (Outcome, error) {
			return CounterValue(u.Balance), nil
		}),
	)

	const goroutines = 32
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)

	for i := range goroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			want := float64(i + 1)
			report, err := EvaluateMetricsWithData(context.Background(), tree, ProcessingHooks{}, "check", metricUser{Balance: want})
			if err != nil {
				errs <- err
				return
			}
			if got := report.Metrics["balance"].Count; got != want {
				errs <- fmt.Errorf("concurrent check failed: got %v, want %v", got, want)
			}
		}(i)
	}

	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent evaluation failed: %v", err)
	}
}

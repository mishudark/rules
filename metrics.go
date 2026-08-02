package rules

import (
	"context"
	"errors"
	"fmt"
	"slices"
)

// Kind identifies the type of metric an outcome carries.
type Kind int

const (
	// KindValid is a boolean outcome (valid/invalid). Use it to report a
	// boolean observation as a metric; pass/fail semantics live in the rule's
	// returned error.
	KindValid Kind = iota
	// KindCounter is a single numeric count (e.g. revenue, page views).
	KindCounter
	// KindHistogram is a distribution of observed values over buckets.
	KindHistogram
	// KindScore is a numeric score, optionally weighted (e.g. risk, health).
	KindScore
)

func (k Kind) String() string {
	switch k {
	case KindValid:
		return "valid"
	case KindCounter:
		return "counter"
	case KindHistogram:
		return "histogram"
	case KindScore:
		return "score"
	default:
		return fmt.Sprintf("kind(%d)", int(k))
	}
}

// Aggregation defines how multiple outcomes that share the same metric name
// are combined when the report is built.
type Aggregation int

const (
	// AggNone uses the kind-specific default (see defaultAggregation).
	AggNone Aggregation = iota
	// AggSum adds values (counter, score).
	AggSum
	// AggMax keeps the largest value.
	AggMax
	// AggMin keeps the smallest value.
	AggMin
	// AggAvg averages the values.
	AggAvg
	// AggWeightedAvg computes a weighted average using each outcome's Weight
	// (score only).
	AggWeightedAvg
	// AggMerge combines histograms bucket-wise (histogram only).
	AggMerge
	// AggLast keeps the last emitted value.
	AggLast
)

// Histogram is a distribution of observed values over le (less-than-or-equal)
// bucket boundaries. Counts are cumulative per boundary, following standard
// Prometheus-style histogram semantics.
type Histogram struct {
	Buckets []float64 // le boundaries, ascending
	Counts  []uint64  // number of observations <= boundary (same length as Buckets)
	Total   uint64    // total number of observations
	Sum     float64   // sum of all observed values
}

// NewHistogram creates an empty histogram with the given le boundaries.
//
// The boundaries are copied and sorted ascending, so the caller's slice is
// never mutated and Observe can use a simple cumulative scan.
//
// Observations strictly greater than the largest boundary are counted in
// Total and Sum but in no bucket; pass math.Inf(1) as the final boundary to
// capture every observation in a bucket.
func NewHistogram(buckets []float64) Histogram {
	buckets = append([]float64(nil), buckets...)
	slices.Sort(buckets)
	return Histogram{
		Buckets: buckets,
		Counts:  make([]uint64, len(buckets)),
	}
}

// Observe records a value into the histogram. Counts are cumulative: every
// bucket whose boundary is >= v is incremented.
func (h *Histogram) Observe(v float64) {
	h.Total++
	h.Sum += v
	for i, boundary := range h.Buckets {
		if v <= boundary {
			h.Counts[i]++
		}
	}
}

// Outcome is a metric value carried by a rule. The value fields that matter
// depend on Kind: Count for KindCounter, Histogram for KindHistogram, Score
// and Weight for KindScore, Valid and Err for KindValid.
type Outcome struct {
	Kind  Kind
	Name  string // metric name, used as the report key
	Field string // label/dimension (e.g. "revenue", "latency_ms")
	Valid bool
	Err   error

	Count     float64
	Histogram Histogram
	Score     float64
	Weight    float64 // weight used by AggWeightedAvg (default 1)

	// Aggregation overrides how this outcome combines with same-name
	// outcomes during report aggregation. AggNone uses the kind default.
	Aggregation Aggregation

	Labels map[string]string // extra fixed dimensions
}

// CounterValue returns a KindCounter outcome carrying a numeric count.
func CounterValue(v float64) Outcome {
	return Outcome{Kind: KindCounter, Count: v}
}

// ScoreValue returns a KindScore outcome carrying a score and an optional
// weight (0 means weight 1).
func ScoreValue(score, weight float64) Outcome {
	return Outcome{Kind: KindScore, Score: score, Weight: weight}
}

// HistogramValue returns a KindHistogram outcome carrying a distribution.
func HistogramValue(h Histogram) Outcome {
	return Outcome{Kind: KindHistogram, Histogram: h}
}

// ValidValue returns a KindValid outcome. A non-nil err is surfaced in the
// report; to also fail the report, return a non-nil error from the rule.
func ValidValue(valid bool, err error) Outcome {
	return Outcome{Kind: KindValid, Valid: valid, Err: err}
}

// outcomeKey is the context key for the outcome collector.
type outcomeKey struct{}

// outcomeCollector gathers the outcomes emitted by rules during the Validate
// phase of EvaluateMetrics. It is a per-evaluation side channel: rules are
// never mutated, so pure metric-carrying rules remain safe to share across
// goroutines.
type outcomeCollector struct {
	outcomes []Outcome
}

// withOutcomeCollector returns a context carrying a fresh outcome collector
// and the collector itself.
func withOutcomeCollector(ctx context.Context) (context.Context, *outcomeCollector) {
	collector := &outcomeCollector{}
	return context.WithValue(ctx, outcomeKey{}, collector), collector
}

// outcomeCollectorFromContext returns the outcome collector attached to ctx,
// or nil when metrics are not being collected.
func outcomeCollectorFromContext(ctx context.Context) *outcomeCollector {
	collector, _ := ctx.Value(outcomeKey{}).(*outcomeCollector)
	return collector
}

func (c *outcomeCollector) add(o Outcome) {
	c.outcomes = append(c.outcomes, o)
}

// Emit records a metric outcome during rule evaluation. Any rule may call
// Emit from its Validate method to carry metric values (counter, histogram,
// score) alongside its pass/fail result. Set the outcome's Name (or Field) to
// control the report key.
//
// When the tree is evaluated with Validate instead of EvaluateMetrics there
// is no collector in the context and Emit is a no-op.
//
// Example:
//
//	rule := rules.NewTypedMetricRule[Order]("itemsInOrder", rules.KindCounter, "items",
//	    func(ctx context.Context, order Order) (rules.Outcome, error) {
//	        return rules.CounterValue(float64(len(order.Items))), nil
//	    })
func Emit(ctx context.Context, o Outcome) {
	if collector := outcomeCollectorFromContext(ctx); collector != nil {
		collector.add(o)
	}
}

// fillOutcome stamps the constructor-declared name, field and kind onto an
// outcome so report aggregation is consistent regardless of how the rule
// function built the value.
func fillOutcome(name, field string, kind Kind, o Outcome) Outcome {
	o.Name = name
	o.Field = field
	o.Kind = kind
	return o
}

// MetricRulePure is a closure-based rule that validates and carries a metric
// outcome. Its data is bound at construction time; it is intended for
// single-use trees.
type MetricRulePure struct {
	RuleBase
	name  string
	kind  Kind
	field string
	fn    func() (Outcome, error)
}

var _ Rule = (*MetricRulePure)(nil)

// Name returns the rule name.
func (r *MetricRulePure) Name() string { return r.name }

// Prepare is a no-op for pure rules.
func (r *MetricRulePure) Prepare(context.Context) (any, error) { return nil, nil }

// Validate executes the wrapped function, records the outcome, and returns
// the validation error.
func (r *MetricRulePure) Validate(ctx context.Context) error {
	if r.fn == nil {
		return Error{
			Field: r.name,
			Err:   "rule function is nil",
			Code:  "RULE_FUNC_NIL",
		}
	}
	outcome, err := r.fn()
	Emit(ctx, fillOutcome(r.name, r.field, r.kind, outcome))
	return err
}

// NewMetricRulePure creates a closure-based rule that carries a metric
// outcome. Data is bound at construction time; use NewMetricRule or
// NewTypedMetricRule for reusable trees.
//
// Example:
//
//	rule := rules.NewMetricRulePure("mrr", rules.KindCounter, "mrr", func() (rules.Outcome, error) {
//	    return rules.CounterValue(1250.5), nil
//	})
func NewMetricRulePure(name string, kind Kind, field string, fn func() (Outcome, error)) Rule {
	return &MetricRulePure{name: name, kind: kind, field: field, fn: fn}
}

// TypedMetricRule is a pure metric-carrying rule that reads data of type T
// from the data registry at validation time. Used by NewTypedMetricRule.
type TypedMetricRule[T any] struct {
	RuleBase
	name  string
	kind  Kind
	field string
	fn    func(ctx context.Context, data T) (Outcome, error)
}

var _ Rule = (*TypedMetricRule[any])(nil)

// Name returns the rule name.
func (r *TypedMetricRule[T]) Name() string { return r.name }

// Prepare is a no-op for pure rules.
func (r *TypedMetricRule[T]) Prepare(context.Context) (any, error) { return nil, nil }

// Validate reads the typed data from the data registry via GetAs[T], runs the
// wrapped function, records the outcome, and returns the validation error.
func (r *TypedMetricRule[T]) Validate(ctx context.Context) error {
	data, ok := GetAs[T](ctx)
	if !ok {
		var zero T
		raw, _ := Get(ctx)
		return Error{
			Field: r.name,
			Err:   fmt.Sprintf("expected data of type %T, got %T", zero, raw),
			Code:  ErrorCodeTypeMismatch,
		}
	}
	if r.fn == nil {
		return Error{
			Field: r.name,
			Err:   "rule function is nil",
			Code:  ErrorCodeRuleFuncNil,
		}
	}
	outcome, err := r.fn(ctx, data)
	Emit(ctx, fillOutcome(r.name, r.field, r.kind, outcome))
	return err
}

// NewTypedMetricRule creates a type-safe rule that reads data of type T from
// the data registry, validates it, and carries a metric outcome. Returns a
// TYPE_MISMATCH error if the registered data is not of type T.
//
// Example:
//
//	rule := rules.NewTypedMetricRule[User]("engagement", rules.KindScore, "engagement",
//	    func(ctx context.Context, u User) (rules.Outcome, error) {
//	        return rules.ScoreValue(u.Engagement, 1), nil
//	    })
func NewTypedMetricRule[T any](name string, kind Kind, field string, fn func(ctx context.Context, data T) (Outcome, error)) Rule {
	return &TypedMetricRule[T]{name: name, kind: kind, field: field, fn: fn}
}

// TypedMetricRuleDataFunc is a rule with Prepare support, type-safe data
// access, and a carried metric outcome. In is the input type read from the
// data registry; T is the loaded data type retrieved during Prepare. The rule
// keeps no state: Prepare records its retrieved data in the per-evaluation
// preparedStore keyed by this rule, and Validate reads it back typed via
// GetPreparedAs[T]. A single tree built with it can be reused and shared
// across goroutines.
type TypedMetricRuleDataFunc[In any, T any] struct {
	RuleBase
	name    string
	kind    Kind
	field   string
	prepare func(ctx context.Context, input In) (T, error)
	fn      func(ctx context.Context, input In, data T) (Outcome, error)
}

var _ Rule = (*TypedMetricRuleDataFunc[any, any])(nil)

// Name returns the rule name.
func (r *TypedMetricRuleDataFunc[In, T]) Name() string { return r.name }

// Prepare reads the typed input from the data registry, runs the prepare
// function, and records the retrieved data in the per-evaluation preparedStore
// keyed by this rule. The rule keeps no state.
func (r *TypedMetricRuleDataFunc[In, T]) Prepare(ctx context.Context) (any, error) {
	input, ok := GetAs[In](ctx)
	if !ok {
		var zero In
		return nil, Error{
			Field: r.name,
			Err:   fmt.Sprintf("expected input of type %T, got different type", zero),
			Code:  ErrorCodeTypeMismatch,
		}
	}

	if r.prepare == nil {
		var zero T
		recordPrepared(ctx, r, zero)
		return zero, nil
	}

	data, err := r.prepare(ctx, input)
	if err != nil {
		return nil, err
	}
	recordPrepared(ctx, r, data)
	return data, nil
}

// Validate reads the typed input from the data registry and the prepared data
// (of type T) recorded by Prepare, runs the validation function, records the
// outcome, and returns the validation error.
func (r *TypedMetricRuleDataFunc[In, T]) Validate(ctx context.Context) error {
	input, ok := GetAs[In](ctx)
	if !ok {
		var zero In
		return Error{
			Field: r.name,
			Err:   fmt.Sprintf("expected input of type %T, got different type", zero),
			Code:  ErrorCodeTypeMismatch,
		}
	}

	loaded, ok := GetPreparedAs[T](ctx, r)
	if !ok {
		return Error{
			Field: r.name,
			Err:   "metric data from prepare not available",
			Code:  ErrorCodeDataNotPrepared,
		}
	}

	outcome, err := r.fn(ctx, input, loaded)
	Emit(ctx, fillOutcome(r.name, r.field, r.kind, outcome))
	return err
}

// NewTypedMetricRuleWithPrepare creates a type-safe rule with Prepare support
// that carries a metric outcome. This is useful when a metric needs to fetch
// additional data or perform side effects before validation (e.g., querying a
// database), and it participates in the same dataloader batching as rules and
// conditions.
//
// Prepare records the retrieved data in the per-evaluation preparedStore
// keyed by this rule; Validate reads it back typed via GetPreparedAs[T]. The
// rule keeps no state, so a tree built with it can be reused and shared across
// goroutines.
//
// Example:
//
//	rule := rules.NewTypedMetricRuleWithPrepare[User, StoredData](
//	    "engagementScore", rules.KindScore, "engagement",
//	    func(ctx context.Context, u User) (StoredData, error) {
//	        return loader.Load(ctx, u.ID) // batched with all other Prepare fetches
//	    },
//	    func(ctx context.Context, u User, d StoredData) (rules.Outcome, error) {
//	        return rules.ScoreValue(d.Score, 1), nil
//	    })
func NewTypedMetricRuleWithPrepare[In any, T any](name string, kind Kind, field string,
	prepare func(ctx context.Context, input In) (T, error),
	fn func(ctx context.Context, input In, data T) (Outcome, error),
) Rule {
	return &TypedMetricRuleDataFunc[In, T]{
		name:    name,
		kind:    kind,
		field:   field,
		prepare: prepare,
		fn:      fn,
	}
}

// Report is the aggregated result of evaluating a tree whose rules may carry
// metric outcomes.
//
// An outcome's error is reported twice by design: once on the metric itself
// (Metrics[name].Err) and once in Errors. Callers that only care about
// pass/fail should use Valid; callers that need details can read either
// location.
type Report struct {
	// Valid is true when no errors were produced by any rule.
	Valid bool
	// Errors contains rule validation errors and outcome errors.
	Errors []error
	// Metrics holds the aggregated outcome of every emitted metric, keyed by
	// metric name.
	Metrics map[string]Outcome
}

// defaultAggregation returns the kind-specific default aggregation.
func defaultAggregation(k Kind) Aggregation {
	switch k {
	case KindCounter:
		return AggSum
	case KindHistogram:
		return AggMerge
	case KindScore:
		return AggWeightedAvg
	default:
		return AggLast
	}
}

// aggregateOutcomes groups emitted outcomes by name and combines each group
// using the aggregation carried by its first outcome. Outcomes that carry
// neither a Name nor a Field have no report slot and are dropped; their
// errors, if any, are still surfaced by the driver.
func aggregateOutcomes(outcomes []Outcome) Report {
	report := Report{Metrics: make(map[string]Outcome, len(outcomes))}

	groups := make(map[string][]Outcome, len(outcomes))
	order := make([]string, 0, len(outcomes))
	for _, o := range outcomes {
		key := o.Name
		if key == "" {
			key = o.Field
		}
		if key == "" {
			continue
		}
		if _, seen := groups[key]; !seen {
			order = append(order, key)
		}
		groups[key] = append(groups[key], o)
	}

	for _, key := range order {
		report.Metrics[key] = finalizeGroup(groups[key])
	}

	return report
}

// finalizeGroup combines every outcome that shares a metric name into a single
// aggregated outcome. The group is guaranteed to be non-empty.
func finalizeGroup(group []Outcome) Outcome {
	res := group[0]
	if len(group) == 1 {
		return res
	}

	agg := res.Aggregation
	if agg == AggNone {
		agg = defaultAggregation(res.Kind)
	}

	switch res.Kind {
	case KindCounter:
		res = aggregateNumeric(group, res, agg, func(o Outcome) float64 { return o.Count })
	case KindScore:
		switch agg {
		case AggWeightedAvg, AggAvg:
			var sum, weights float64
			for _, o := range group {
				w := o.Weight
				if w == 0 {
					w = 1
				}
				sum += o.Score * w
				weights += w
			}
			res.Score = sum / weights
		case AggSum:
			var total float64
			for _, o := range group {
				total += o.Score
			}
			res.Score = total
		case AggMax:
			res = aggregateNumeric(group, res, agg, func(o Outcome) float64 { return o.Score })
		case AggMin:
			res = aggregateNumeric(group, res, agg, func(o Outcome) float64 { return o.Score })
		case AggLast:
			res = group[len(group)-1]
		}
	case KindHistogram:
		res.Histogram = mergeHistograms(group)
	case KindValid:
		res.Valid = true
		for _, o := range group {
			if !o.Valid {
				res.Valid = false
				break
			}
		}
	default:
		res = group[len(group)-1]
	}

	res.Err = joinOutcomeErrors(group)
	return res
}

// aggregateNumeric combines a numeric outcome kind (counter or score) using
// the given aggregation and value accessor.
func aggregateNumeric(group []Outcome, res Outcome, agg Aggregation, value func(Outcome) float64) Outcome {
	switch agg {
	case AggSum:
		var total float64
		for _, o := range group {
			total += value(o)
		}
		switch res.Kind {
		case KindCounter:
			res.Count = total
		case KindScore:
			res.Score = total
		}
	case AggAvg:
		var total float64
		for _, o := range group {
			total += value(o)
		}
		avg := total / float64(len(group))
		switch res.Kind {
		case KindCounter:
			res.Count = avg
		case KindScore:
			res.Score = avg
		}
	case AggMax:
		for _, o := range group {
			if value(o) > value(res) {
				res = o
			}
		}
	case AggMin:
		for _, o := range group {
			if value(o) < value(res) {
				res = o
			}
		}
	case AggLast:
		res = group[len(group)-1]
	}
	return res
}

// mergeHistograms combines histograms. Total and Sum always accumulate. Bucket
// counts are summed only for boundaries that exactly match the winning boundary
// set (the first non-empty histogram's), so a histogram with a different
// boundary layout contributes its Total and Sum but no bucket counts instead of
// silently merging counts into the wrong bucket.
func mergeHistograms(group []Outcome) Histogram {
	res := group[0].Histogram
	for _, o := range group[1:] {
		h := o.Histogram
		res.Total += h.Total
		res.Sum += h.Sum

		if len(res.Buckets) == 0 && len(h.Buckets) > 0 {
			res.Buckets = append([]float64(nil), h.Buckets...)
			res.Counts = make([]uint64, len(h.Buckets))
		}

		for i, boundary := range res.Buckets {
			if idx := boundaryIndex(h.Buckets, boundary); idx >= 0 {
				res.Counts[i] += h.Counts[idx]
			}
		}
	}
	return res
}

// boundaryIndex returns the index of boundary in buckets, or -1 when absent.
func boundaryIndex(buckets []float64, boundary float64) int {
	for i, b := range buckets {
		if b == boundary {
			return i
		}
	}
	return -1
}

// joinOutcomeErrors joins all non-nil errors carried by the group.
func joinOutcomeErrors(group []Outcome) error {
	var errs []error
	for _, o := range group {
		if o.Err != nil {
			errs = append(errs, o.Err)
		}
	}
	return errors.Join(errs...)
}

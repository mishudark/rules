package rules

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// eventLog records engine lifecycle events in order, simulating how a
// dataloader would observe prepare/validate calls.
type eventLog struct {
	events []string
}

func (l *eventLog) add(format string, args ...any) {
	l.events = append(l.events, fmt.Sprintf(format, args...))
}

// indexOf returns the index of the first event with the given prefix, or -1.
func (l *eventLog) indexOf(prefix string) int {
	for i, e := range l.events {
		if strings.HasPrefix(e, prefix) {
			return i
		}
	}
	return -1
}

// lastIndexOf returns the index of the last event with the given prefix, or -1.
func (l *eventLog) lastIndexOf(prefix string) int {
	for i := len(l.events) - 1; i >= 0; i-- {
		if strings.HasPrefix(l.events[i], prefix) {
			return i
		}
	}
	return -1
}

// count returns the number of events with the given prefix.
func (l *eventLog) count(prefix string) int {
	n := 0
	for _, e := range l.events {
		if strings.HasPrefix(e, prefix) {
			n++
		}
	}
	return n
}

type loggingCondition struct {
	name  string
	log   *eventLog
	valid bool
	pure  bool
}

func (c *loggingCondition) Name() string { return c.name }
func (c *loggingCondition) Prepare(ctx context.Context) (any, error) {
	c.log.add("prepareCondition:%s", c.name)
	return nil, nil
}
func (c *loggingCondition) IsValid(ctx context.Context) bool {
	c.log.add("isValid:%s", c.name)
	return c.valid
}
func (c *loggingCondition) IsPure() bool { return c.pure }

type loggingRule struct {
	name string
	log  *eventLog
}

func (r *loggingRule) Name() string                 { return r.name }
func (r *loggingRule) SetExecutionPath(path string) {}
func (r *loggingRule) GetExecutionPath() string     { return "" }
func (r *loggingRule) Prepare(ctx context.Context) (any, error) {
	r.log.add("prepareRule:%s", r.name)
	return nil, nil
}
func (r *loggingRule) Validate(ctx context.Context) error {
	r.log.add("validateRule:%s", r.name)
	return nil
}

// assertPhaseOrdering verifies the dataloader architecture invariant:
//  1. every condition Prepare happens before any rule Prepare
//  2. every rule Prepare happens before any rule Validate
func assertPhaseOrdering(t *testing.T, log *eventLog) {
	t.Helper()

	lastCondPrepare := log.lastIndexOf("prepareCondition:")
	firstRulePrepare := log.indexOf("prepareRule:")
	lastRulePrepare := log.lastIndexOf("prepareRule:")
	firstValidate := log.indexOf("validateRule:")

	if firstRulePrepare == -1 || firstValidate == -1 {
		t.Fatalf("incomplete event log: %v", log.events)
	}

	if lastCondPrepare != -1 && lastCondPrepare > firstRulePrepare {
		t.Errorf("a condition was prepared after a rule prepare started; events: %v", log.events)
	}
	if lastRulePrepare > firstValidate {
		t.Errorf("a rule was prepared after validation started; events: %v", log.events)
	}
}

// TestArchitecture_ValidatePhaseOrdering builds a tree exercising every node
// type with impure conditions and asserts the 4-phase ordering: all condition
// prepares, then evaluation, then all rule prepares, then all validates.
func TestArchitecture_ValidatePhaseOrdering(t *testing.T) {
	t.Parallel()

	log := &eventLog{}

	condA := &loggingCondition{name: "A", log: log, valid: true}
	condB := &loggingCondition{name: "B", log: log, valid: false} // impure false: children still prepared
	condC := &loggingCondition{name: "C", log: log, valid: true}
	rule1 := &loggingRule{name: "rule1", log: log}
	rule2 := &loggingRule{name: "rule2", log: log}

	tree := Root(
		Node(condA,
			AllOf(
				Rules(rule1),
				Either(condC,
					[]Evaluable{Rules()},
					[]Evaluable{Rules()},
				),
			),
		),
		Node(condB, Rules(&loggingRule{name: "unreachable", log: log})),
		Rules(rule2),
	)

	if err := Validate(context.Background(), tree, ProcessingHooks{}, "arch"); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	assertPhaseOrdering(t, log)

	// Impure conditions must be prepared even when they evaluate to false,
	// so a dataloader can batch their fetches with the rest of the tree.
	if log.count("prepareCondition:B") != 1 {
		t.Errorf("impure false condition B should be prepared exactly once; events: %v", log.events)
	}

	// No impure condition may be evaluated (IsValid) before every condition
	// in the tree has been prepared: evaluation is a separate phase.
	lastCondPrepare := log.lastIndexOf("prepareCondition:")
	if firstIsValid := log.indexOf("isValid:"); firstIsValid != -1 && firstIsValid < lastCondPrepare {
		t.Errorf("an impure condition was evaluated before all conditions were prepared; events: %v", log.events)
	}

	// Rules behind the false condition must not be prepared or validated.
	if log.count("prepareRule:unreachable") != 0 || log.count("validateRule:unreachable") != 0 {
		t.Errorf("rules behind a false condition must not run; events: %v", log.events)
	}

	// Reached rules must be prepared exactly once and validated exactly once.
	for _, name := range []string{"rule1", "rule2"} {
		if log.count("prepareRule:"+name) != 1 {
			t.Errorf("%s should be prepared exactly once; events: %v", name, log.events)
		}
		if log.count("validateRule:"+name) != 1 {
			t.Errorf("%s should be validated exactly once; events: %v", name, log.events)
		}
	}
}

// TestArchitecture_ValidateMultiPhaseOrdering asserts that across multiple
// targets, ALL targets' conditions are prepared before ANY rule is prepared,
// and ALL rules are prepared before ANY validation. This is what lets a
// dataloader batch fetches across an entire batch of inputs.
func TestArchitecture_ValidateMultiPhaseOrdering(t *testing.T) {
	t.Parallel()

	log := &eventLog{}

	buildTarget := func(suffix string) Target {
		cond := &loggingCondition{name: "cond" + suffix, log: log, valid: true}
		rule := &loggingRule{name: "rule" + suffix, log: log}
		return *NewTarget(context.Background(), Root(Node(cond, Rules(rule))))
	}

	targets := []Target{buildTarget("1"), buildTarget("2"), buildTarget("3")}

	if err := ValidateMulti(context.Background(), targets, ProcessingHooks{}, "arch"); err != nil {
		t.Fatalf("ValidateMulti failed: %v", err)
	}

	assertPhaseOrdering(t, log)

	// All three conditions prepared before the first rule prepare.
	lastCondPrepare := log.lastIndexOf("prepareCondition:")
	if log.count("prepareCondition:") != 3 {
		t.Errorf("expected 3 condition prepares; events: %v", log.events)
	}
	if firstRulePrepare := log.indexOf("prepareRule:"); firstRulePrepare < lastCondPrepare {
		t.Errorf("rule prepares started before all targets' conditions were prepared; events: %v", log.events)
	}
}

// TestArchitecture_ImpureEitherPreparesBothBranches verifies the dataloader
// batching invariant for ConditionEither: with an impure condition, both
// branches' conditions are prepared even though only one branch can match.
func TestArchitecture_ImpureEitherPreparesBothBranches(t *testing.T) {
	t.Parallel()

	log := &eventLog{}
	eitherCond := &loggingCondition{name: "either", log: log, valid: true}
	leftCond := &loggingCondition{name: "left", log: log, valid: true}
	rightCond := &loggingCondition{name: "right", log: log, valid: true}

	tree := Root(
		Either(eitherCond,
			[]Evaluable{Node(leftCond, Rules(&loggingRule{name: "leftRule", log: log}))},
			[]Evaluable{Node(rightCond, Rules(&loggingRule{name: "rightRule", log: log}))},
		),
	)

	if err := Validate(context.Background(), tree, ProcessingHooks{}, "arch"); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	if log.count("prepareCondition:left") != 1 {
		t.Errorf("left branch condition should be prepared; events: %v", log.events)
	}
	if log.count("prepareCondition:right") != 1 {
		t.Errorf("right branch condition should ALSO be prepared for dataloader batching; events: %v", log.events)
	}

	// Only the matching (left) branch's rules run.
	if log.count("validateRule:leftRule") != 1 || log.count("validateRule:rightRule") != 0 {
		t.Errorf("only the left branch rules should be validated; events: %v", log.events)
	}
}

// TestArchitecture_NilEitherConditionPreparesRightBranch is a regression
// test: Evaluate treats a nil Either condition as false and selects the
// right branch, so PrepareConditions must prepare the right branch too.
// Previously it prepared nothing, so right-branch rules were validated
// while their subtree conditions had never been prepared.
func TestArchitecture_NilEitherConditionPreparesRightBranch(t *testing.T) {
	t.Parallel()

	log := &eventLog{}
	rightCond := &loggingCondition{name: "right", log: log, valid: true}

	tree := Root(
		Either(nil,
			[]Evaluable{Node(&loggingCondition{name: "left", log: log, valid: true},
				Rules(&loggingRule{name: "leftRule", log: log}))},
			[]Evaluable{Node(rightCond, Rules(&loggingRule{name: "rightRule", log: log}))},
		),
	)

	if err := Validate(context.Background(), tree, ProcessingHooks{}, "arch"); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	if log.count("prepareCondition:right") != 1 {
		t.Errorf("right branch condition must be prepared when condition is nil; events: %v", log.events)
	}
	if log.count("prepareCondition:left") != 0 {
		t.Errorf("left branch must not be prepared when condition is nil; events: %v", log.events)
	}
	if log.count("validateRule:rightRule") != 1 {
		t.Errorf("right branch rule should be validated; events: %v", log.events)
	}

	assertPhaseOrdering(t, log)
}

// TestArchitecture_CompositeRulesPrepareEverything verifies that composite
// rules (Or, ChainRules) prepare ALL of their children during the prepare
// phase, before any Validate runs: preparation is setup work and must not
// follow the short-circuit semantics of validation.
func TestArchitecture_CompositeRulesPrepareEverything(t *testing.T) {
	t.Parallel()

	log := &eventLog{}

	orChild1 := &loggingRule{name: "orChild1", log: log}
	orChild2 := &loggingRule{name: "orChild2", log: log}
	chainChild1 := &loggingRule{name: "chainChild1", log: log}
	chainChild2 := &loggingRule{name: "chainChild2", log: log}

	tree := Root(Rules(
		Or(orChild1, orChild2),
		NewChainRules(chainChild1, chainChild2),
	))

	if err := Validate(context.Background(), tree, ProcessingHooks{}, "arch"); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	for _, name := range []string{"orChild1", "orChild2", "chainChild1", "chainChild2"} {
		if log.count("prepareRule:"+name) != 1 {
			t.Errorf("composite child %s should be prepared exactly once; events: %v", name, log.events)
		}
	}

	assertPhaseOrdering(t, log)
}

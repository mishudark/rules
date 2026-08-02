package rules

import (
	"context"
	"errors"
	"fmt"
)

// Error contains a structured definition for validation errors, including
// the field related to the error, a descriptive error message, and an
// optional error code for easier identification or localization.
type Error struct {
	Field string // Field indicates the specific input field or area where the error occurred.
	Err   string // Err provides a human-readable description of the error.
	Code  string // Code is an optional identifier for the type of error.
}

// Error implements the standard Go error interface, providing a formatted
// string representation of the validation error details.
func (e Error) Error() string {
	return fmt.Sprintf("code: %s, field: %s, error: %s", e.Code, e.Field, e.Err)
}

// Condition represents a function that evaluates to true or false, typically
// used within conditional nodes (like ConditionNode) to determine whether
// associated rules or child nodes should be processed.
//
// Prepare returns the retrieved data (any) and an error; the implementation
// records that data itself in the per-evaluation preparedStore (see
// [PutPrepared]) so it can be read back typed in IsValid via [GetPreparedAs].
// Because the prepared data travels in the context — never on the condition —
// a single tree can be reused and shared across goroutines.
type Condition interface {
	// Prepare is executed before the main validation logic. It can be used to
	// retrieve information. It returns the retrieved data (any) and an error;
	// for typed conditions the same data is recorded in the preparedStore so
	// IsValid can read it back via [GetPreparedAs].
	Prepare(ctx context.Context) (any, error)
	// Name is a method to retrieve the name of the condition for debugging or logging.
	Name() string
	// Evaluate returns true if the condition is met, otherwise false. Typed
	// conditions read back the data Prepare recorded via [GetPreparedAs]; pure
	// conditions may ignore it.
	IsValid(ctx context.Context) bool
	// IsPure returns true if the condition is pure (no side effects).
	IsPure() bool
}

// Rule represents a single unit of validation logic. It includes a Prepare
// step (potentially for setup or pre-checks) and a Validate step that performs
// the actual validation check.
//
// Prepare returns the retrieved data (any) plus an error; the implementation
// records that data in the per-evaluation preparedStore (see [PutPrepared]) so
// it can be read back typed in Validate via [GetPreparedAs]. Rules never store
// prepared state on themselves, so a single tree can be reused and shared
// across goroutines.
type Rule interface {
	// Name returns the name of the rule for identification.
	Name() string
	// Prepare allows for initialization or pre-checks before the main validation.
	// It returns the retrieved data and an error if preparation fails. Typed
	// rules record the same data in the preparedStore so Validate can read it
	// back via [GetPreparedAs].
	Prepare(ctx context.Context) (any, error)
	// Validate performs the core validation logic. Typed rules read back the
	// data Prepare recorded via [GetPreparedAs] (typed); pure rules may ignore
	// it. Returns an error if validation fails, otherwise nil.
	Validate(ctx context.Context) error
}

// Evaluable represents any component (like a node or a set of rules) within the
// validation structure that can be evaluated. The evaluation determines if the
// component's conditions are met (returning true) and provides the list of Rules
// that should be executed as a result.
type Evaluable interface {
	// PrepareConditions is executed before the main validation logic. It can be used to retrieve information.
	// specifically its target are the children condition nodes.
	PrepareConditions(ctx context.Context) error
	// Evaluate checks the conditions of the component and returns whether it
	// passes (bool) and the list of Rules associated with it if it passes.
	// If the conditions are not met, it returns false and a nil slice of Rules.
	Evaluate(ctx context.Context, executionPath string) (bool, []Rule)
}

// LeafNode represents a terminal node in the validation evaluation tree.
// It directly contains a slice of Rules that should be executed if this
// node is reached and evaluated successfully.
type LeafNode struct {
	Rules []Rule
}

// PrepareConditions is a no-op for LeafNode. It always returns nil.
func (r *LeafNode) PrepareConditions(ctx context.Context) error {
	// LeafNode does not have conditions to prepare.
	return nil
}

// Evaluate implements the Evaluable interface for LeafNode. It always
// returns true, indicating success, along with the slice of Rules contained
// within the node.
//
// If the context carries an ExecutionTrace (see WithExecutionTrace), the
// path of each rule is recorded in the trace. Rules are never mutated, so a
// LeafNode is safe for concurrent evaluation.
func (n *LeafNode) Evaluate(ctx context.Context, executionPath string) (bool, []Rule) {
	if trace := traceFromContext(ctx); trace != nil {
		for _, rule := range n.Rules {
			trace.record(rule, fmt.Sprintf("%s -> %s -> %s", executionPath, "leafNode", rule.Name()))
		}
	}

	return true, n.Rules
}

var _ Evaluable = (*LeafNode)(nil) // Ensure LeafNode implements the Evaluable interface.

// ConditionNode represents a node in the validation evaluation tree that has an
// associated Condition. If the Condition evaluates to true, the ConditionNode
// then evaluates its child Evaluables, accumulating the Rules from those children
// that also evaluate successfully.
type ConditionNode struct {
	Condition  Condition   // The condition that must be true for children to be evaluated.
	Evaluables []Evaluable // The child nodes or rule sets to evaluate if Condition is true.
}

// PrepareConditions prepares the ConditionNode's condition and recursively
// prepares its children.
//
// Optimization: when the condition is pure (no side effects), it can be
// evaluated immediately, and if it is false the children are skipped —
// nothing down this branch will run.
//
// For impure conditions, the children are ALWAYS prepared, even when the
// condition will turn out to be false. This is intentional: impure
// Prepare() calls are expected to fan out data fetches that a dataloader
// will batch and deduplicate (single round-trip for the whole tree).
// Short-circuiting here would serialize those fetches across branches,
// producing N+1 round-trips. The IsValid check is deferred to Evaluate.
func (n *ConditionNode) PrepareConditions(ctx context.Context) error {
	if n.Condition == nil {
		return nil
	}

	// If the condition is pure, evaluate it immediately and short-circuit
	// the subtree when it cannot pass. Pure conditions ignore prepared data.
	if n.Condition.IsPure() && !n.Condition.IsValid(ctx) {
		return nil
	}

	if _, err := n.Condition.Prepare(ctx); err != nil {
		return err
	}

	for _, evaluable := range n.Evaluables {
		if err := evaluable.PrepareConditions(ctx); err != nil {
			return err
		}
	}

	return nil
}

// Evaluate implements the Evaluable interface for ConditionNode. It first checks
// the Condition. If the Condition is nil or evaluates to false, Evaluate returns
// false and nil rules. If the Condition is true, it evaluates each child Evaluable,
// collecting and returning all Rules from children that evaluate successfully (return true).
func (n *ConditionNode) Evaluate(ctx context.Context, executionPath string) (bool, []Rule) {
	if n.Condition == nil || !n.Condition.IsValid(ctx) {
		return false, nil
	}

	matchRules := []Rule{}

	for _, evaluable := range n.Evaluables {
		ok, rules := evaluable.Evaluate(ctx, fmt.Sprintf("%s -> %s", executionPath, n.Condition.Name()))
		if ok {
			matchRules = append(matchRules, rules...)
		}
	}

	// The ConditionNode itself succeeded because its condition was met.
	// It returns the aggregated rules from its successful children.
	return true, matchRules
}

var _ Evaluable = (*ConditionNode)(nil) // Ensure ConditionNode implements the Evaluable interface.

// AllOfNode represents a logical AND operation in the validation evaluation tree.
// All of its child Evaluables must evaluate successfully for the AllOfNode itself
// to be considered successful.
type AllOfNode struct {
	Children []Evaluable // The children that must all evaluate successfully.
}

// PrepareConditions for AllOfNode.
func (n *AllOfNode) PrepareConditions(ctx context.Context) error {
	for _, child := range n.Children {
		err := child.PrepareConditions(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// Evaluate implements the Evaluable interface for AllOfNode. It iterates through
// all its Children. If any child evaluates to false, the AllOfNode immediately
// returns false and nil rules. If all children evaluate to true, it returns true
// and the combined list of Rules gathered from all children. An empty AllOfNode
// is considered successful.
func (n *AllOfNode) Evaluate(ctx context.Context, executionPath string) (bool, []Rule) {
	acc := []Rule{}

	if len(n.Children) == 0 {
		return true, acc // An empty AND condition is trivially true.
	}

	for i := range n.Children {
		child := n.Children[i]
		ok, rules := child.Evaluate(ctx, fmt.Sprintf("%s -> %s", executionPath, "allOfNode"))
		if ok {
			acc = append(acc, rules...)
		} else {
			// If any child fails, the AND condition fails.
			return false, nil
		}
	}

	// All children succeeded.
	return true, acc
}

var _ Evaluable = (*AllOfNode)(nil) // Ensure AllOfNode implements the Evaluable interface.

// AnyOfNode represents a logical OR operation in the validation evaluation tree.
// At least one of its child Evaluables must evaluate successfully for the AnyOfNode
// itself to be considered successful.
type AnyOfNode struct {
	name     string      // Name of the AnyOfNode (optional) for identification or debugging.
	Children []Evaluable // The children, where at least one must evaluate successfully.
}

// PrepareConditions for AnyOfNode.
func (n *AnyOfNode) PrepareConditions(ctx context.Context) error {
	for _, child := range n.Children {
		err := child.PrepareConditions(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// Evaluate implements the Evaluable interface for AnyOfNode. It iterates through
// all its Children. If at least one child evaluates to true, the AnyOfNode returns
// true along with the combined list of Rules gathered from *all* successful
// children. If no children evaluate to true, it returns false and nil rules.
// An empty AnyOfNode is considered successful (or perhaps should be false, depending on desired logic - current impl returns true).
func (n *AnyOfNode) Evaluate(ctx context.Context, executionPath string) (bool, []Rule) {
	acc := []Rule{}

	if len(n.Children) == 0 {
		// Current implementation returns true, similar to AllOfNode.
		return true, acc
	}

	var anyOk bool

	nodeName := n.name
	if nodeName == "" {
		nodeName = "anyOfNode"
	}

	for i := range n.Children {
		child := n.Children[i]
		ok, rules := child.Evaluate(ctx, fmt.Sprintf("%s -> %s", executionPath, nodeName))
		if ok {
			anyOk = true
			acc = append(acc, rules...) // Collect rules from all successful children.
		}
	}

	if !anyOk {
		// No child succeeded.
		return false, nil
	}

	// At least one child succeeded.
	return true, acc
}

var _ Evaluable = (*AnyOfNode)(nil) // Ensure AnyOfNode implements the Evaluable interface.

// AllOf is a constructor function that creates and returns a new AllOfNode
// containing the provided child Evaluables.
func AllOf(children ...Evaluable) Evaluable {
	return &AllOfNode{Children: children}
}

// Rules is a constructor function that creates and returns a new LeafNode
// containing the provided Rules. This is typically used to define the set
// of validations to run at the end of a branch in the evaluation tree.
func Rules(rules ...Rule) Evaluable {
	return &LeafNode{Rules: rules}
}

// Node is a constructor function that creates and returns a new ConditionNode.
// It associates a Condition with a set of child Evaluables.
func Node(condition Condition, children ...Evaluable) Evaluable {
	return &ConditionNode{
		Condition:  condition,
		Evaluables: children,
	}
}

// AnyOf is a constructor function that creates and returns a new AnyOfNode
// containing the provided child Evaluables.
func AnyOf(children ...Evaluable) Evaluable {
	return &AnyOfNode{Children: children}
}

// Root is a constructor function often used to define the top-level node of
// the validation evaluation tree. Currently, it creates an AnyOfNode, implying the
// root requires at least one of its top-level children to evaluate successfully.
func Root(children ...Evaluable) Evaluable {
	// Note: Currently identical to AnyOf().
	return &AnyOfNode{Children: children, name: "root"}
}

type NotCondition struct {
	condition Condition
}

func (n *NotCondition) Name() string {
	if n.condition == nil {
		return "Not -> nil"
	}
	return fmt.Sprintf("Not -> %s", n.condition.Name())
}

func (n *NotCondition) Prepare(ctx context.Context) (any, error) {
	if n.condition == nil {
		return nil, nil
	}
	return n.condition.Prepare(ctx)
}

func (n *NotCondition) IsValid(ctx context.Context) bool {
	if n.condition == nil {
		// Avoid nil pointer dereference if Condition func wasn't provided.
		return false
	}

	return !n.condition.IsValid(ctx)
}

func (n *NotCondition) IsPure() bool {
	if n.condition == nil {
		return false
	}
	return n.condition.IsPure()
}

var _ Condition = (*NotCondition)(nil) // Ensure NotCondition implements the Condition interface.

// ConditionEither represents a node in the validation evaluation tree that has an
// associated Condition. If the Condition evaluates to true, ConditionEither
// evaluates its left Evaluables and returns their rules. If the Condition
// evaluates to false, it evaluates its right Evaluables instead.
type ConditionEither struct {
	Condition Condition   // The condition that determines which branch to evaluate.
	Left      []Evaluable // The evaluables to use if condition is true.
	Right     []Evaluable // The evaluables to use if condition is false.
}

// PrepareConditions prepares the ConditionEither's condition and the
// appropriate branch's children.
//
// Optimization: when the condition is pure (no side effects), it can be
// evaluated immediately and only the matching branch is prepared.
//
// For impure conditions, BOTH branches are always prepared. This is
// intentional: impure Prepare() calls are expected to fan out data fetches
// that a dataloader will batch and deduplicate (single round-trip for the
// whole tree). Skipping the non-matching branch's children would serialize
// fetches across branches, producing N+1 round-trips. The IsValid check
// is deferred to Evaluate.
func (n *ConditionEither) PrepareConditions(ctx context.Context) error {
	if n.Condition == nil {
		// Evaluate treats a nil condition as false and selects the right
		// branch, so prepare that branch here to keep the
		// prepare-all-conditions-first invariant.
		for _, evaluable := range n.Right {
			if err := evaluable.PrepareConditions(ctx); err != nil {
				return err
			}
		}
		return nil
	}

	// Pure: evaluate immediately and prepare only the matching branch.
	// Pure conditions ignore prepared data.
	if n.Condition.IsPure() {
		var sideToPrepare []Evaluable
		if n.Condition.IsValid(ctx) {
			sideToPrepare = n.Left
		} else {
			sideToPrepare = n.Right
		}

		for _, evaluable := range sideToPrepare {
			if err := evaluable.PrepareConditions(ctx); err != nil {
				return err
			}
		}
		return nil
	}

	// Impure: prepare the condition, then fan out Prepare across BOTH
	// branches so the dataloader can batch all fetches together. The typed
	// condition self-records its prepared data, so the store is populated here.
	if _, err := n.Condition.Prepare(ctx); err != nil {
		return err
	}

	for _, evaluable := range n.Left {
		if err := evaluable.PrepareConditions(ctx); err != nil {
			return err
		}
	}
	for _, evaluable := range n.Right {
		if err := evaluable.PrepareConditions(ctx); err != nil {
			return err
		}
	}

	return nil
}

// Evaluate implements the Evaluable interface for ConditionEither. It checks
// the Condition. If the Condition is true, it evaluates the left Evaluables.
// If the Condition is false or nil, it evaluates the right Evaluables.
func (n *ConditionEither) Evaluate(ctx context.Context, executionPath string) (bool, []Rule) {
	var matchRules []Rule

	if n.Condition != nil && n.Condition.IsValid(ctx) {
		// Condition is true, evaluate left branch
		for _, evaluable := range n.Left {
			ok, rules := evaluable.Evaluate(ctx, fmt.Sprintf("%s -> %s (true)", executionPath, n.Condition.Name()))
			if ok {
				matchRules = append(matchRules, rules...)
			}
		}
	} else {
		// Condition is false or nil, evaluate right branch
		for _, evaluable := range n.Right {
			conditionName := "nil"
			if n.Condition != nil {
				conditionName = n.Condition.Name()
			}
			ok, rules := evaluable.Evaluate(ctx, fmt.Sprintf("%s -> %s (false)", executionPath, conditionName))
			if ok {
				matchRules = append(matchRules, rules...)
			}
		}
	}

	return true, matchRules
}

var _ Evaluable = (*ConditionEither)(nil) // Ensure ConditionEither implements the Evaluable interface.

// Either is a constructor function that creates and returns a new ConditionEither.
// It associates a Condition with left and right Evaluables. If the condition is true,
// the left Evaluables are evaluated; otherwise, the right Evaluables are evaluated.
func Either(condition Condition, left, right []Evaluable) Evaluable {
	return &ConditionEither{
		Condition: condition,
		Left:      left,
		Right:     right,
	}
}

// Not is a helper function that takes a Condition and returns a Condition with
// the logical negation of the Condition's result.
func Not(condition Condition) Condition {
	return &NotCondition{
		condition: condition,
	}
}

// NopRule is intended as a placeholder or no-operation function within validation logic.
// It returns nil, signifying success without performing any action. It can be useful
// in conditional logic where one branch requires no validation or during testing.
type NopRule struct {
	RuleBase
}

func (n *NopRule) Name() string {
	return "nopRule"
}

func (n *NopRule) Prepare(ctx context.Context) (any, error) {
	return nil, nil
}

func (n *NopRule) Validate(ctx context.Context) error {
	return nil
}

var _ Rule = (*NopRule)(nil) // Ensure NopRule implements the Rule interface.

// ChainRules represents a Rule that encapsulates
// a sequence of other Rules. When Prepare or Validate is called on ChainRules,
// it executes the corresponding method on each child Rule in order, stopping
// and returning the first encountered error. If all child rules succeed, it returns nil.
//
// Use ChainRules to create a sequential validation chain.
type ChainRules struct {
	RuleBase
	Rules []Rule
}

// Name returns the rule name.
func (c *ChainRules) Name() string {
	return "chainRules"
}

// Prepare implements the Rule interface for ChainRules. It calls Prepare() on each
// Rule in the sequence. If any child Rule's Prepare() returns an error,
// this method stops and returns that error immediately. If all children's
// Prepare() methods succeed, it returns nil. Each child prepares and records
// its own data (typed rules self-record into the preparedStore).
func (c *ChainRules) Prepare(ctx context.Context) (any, error) {
	for _, rule := range c.Rules {
		if _, err := rule.Prepare(ctx); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

// Validate implements the Rule interface for ChainRules. It calls Validate() on each
// Rule in the sequence. If any child Rule's Validate() returns an error,
// this method stops and returns that error immediately. If all children's
// Validate() methods succeed, it returns nil.
func (c *ChainRules) Validate(ctx context.Context) error {
	for _, rule := range c.Rules {
		if err := rule.Validate(ctx); err != nil {
			return err
		}
	}
	return nil
}

// NewChainRules creates a sequential validation chain from the provided rules.
// The rules are executed in order, stopping and returning the first error encountered.
func NewChainRules(rules ...Rule) Rule {
	return &ChainRules{Rules: rules}
}

// RuleBase provides a basic implementation of the Rule interface.
type RuleBase struct{}

// RulePure provides a basic implementation of the Rule interface by wrapping
// a single function. This function represents the core validation logic.
// The Prepare method for a RulePure is a no-op.
type RulePure struct {
	RuleBase
	name string
	rule func() error
}

var _ Rule = (*RulePure)(nil) // Ensure RulePure implements the Rule interface.

// Prepare implements the Rule interface for RulePure. It performs no action
// and always returns nil data and a nil error.
func (r *RulePure) Prepare(ctx context.Context) (any, error) {
	return nil, nil // Simple rules typically don't require preparation.
}

// Name returns the name of the RulePure. This is useful for debugging.
func (r *RulePure) Name() string {
	return r.name
}

// Validate implements the Rule interface for RulePure. It executes the
// wrapped Rule function and returns its result (error or nil).
func (r *RulePure) Validate(ctx context.Context) error {
	if r.rule == nil {
		return Error{
			Field: r.name,
			Err:   "rule function is nil",
			Code:  "RULE_FUNC_NIL",
		}
	}

	return r.rule()
}

// NewRulePure is a constructor function that creates and returns a new RulePure.
func NewRulePure(name string, rule func() error) Rule {
	return &RulePure{
		name: name,
		rule: rule,
	}
}

// OrRules represents a Rule that encapsulates multiple Rules.
// When Prepare or Validate is called on OrRules, it executes the corresponding
// method on each child Rule. If any child Rule succeeds, it stops and returns nil.
// If all child rules fail, it returns the error from the last rule.
type OrRules struct {
	RuleBase
	Rules []Rule
}

var _ Rule = (*OrRules)(nil) // Ensure OrRules implements the Rule interface.

// Name returns a combined name for OrRules.
func (o *OrRules) Name() string {
	return "orRules"
}

// Prepare implements the Rule interface for OrRules. It calls Prepare() on
// every child Rule: preparation is setup work, so all rules must be prepared
// regardless of the OR semantics used by Validate. Each child records its own
// retrieved data into the evaluation's preparedStore. If any child Rule's
// Prepare() returns an error, it stops and returns that error immediately.
func (o *OrRules) Prepare(ctx context.Context) (any, error) {
	for _, rule := range o.Rules {
		if _, err := rule.Prepare(ctx); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

// Validate implements the Rule interface for OrRules. It calls Validate() on each
// Rule. If any child Rule's Validate() returns nil, it returns nil immediately.
// If all rules fail, it returns all errors.
func (o *OrRules) Validate(ctx context.Context) error {
	var errs []error
	if len(o.Rules) == 0 {
		return nil
	}

	for _, rule := range o.Rules {
		err := rule.Validate(ctx)
		// If any rule succeeds, return nil immediately.
		if err == nil {
			return nil
		}
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

// Or is a constructor function that creates and returns a new OrRules.
func Or(rule Rule, rules ...Rule) Rule {
	return &OrRules{
		Rules: append([]Rule{rule}, rules...),
	}
}

// ConditionPure does not need to be prepared and is used as a placeholder
type ConditionPure struct {
	name      string
	condition func() bool
}

var _ Condition = (*ConditionPure)(nil) // Ensure ConditionPure implements the Condition interface.

// Prepare is a no-op for ConditionPure, it always returns nil data and nil error.
func (c *ConditionPure) Prepare(context.Context) (any, error) {
	return nil, nil
}

func (c *ConditionPure) Name() string {
	return c.name
}

func (c *ConditionPure) IsValid(ctx context.Context) bool {
	return c.condition()
}

func (c *ConditionPure) IsPure() bool {
	return true
}

// NewConditionPure is a constructor function that creates and returns a new ConditionPure.
func NewConditionPure(name string, condition func() bool) Condition {
	return &ConditionPure{
		name:      name,
		condition: condition,
	}
}

// ConditionSideEffect has side effects (e.g., fetching data) and uses Prepare.
// The loaded data type T is a type parameter: the prepare function returns a
// value of type T and the condition function receives it typed — no `any`
// conversion at the call site.
type ConditionSideEffect[T any] struct {
	name      string
	prepare   func(ctx context.Context) (T, error)
	condition func(ctx context.Context, data T) bool
}

var _ Condition = (*ConditionSideEffect[any])(nil) // Ensure ConditionSideEffect implements the Condition interface.

// Prepare runs the side-effecting prepare function, records the retrieved data
// in the per-evaluation preparedStore keyed by this condition, and returns it
// (as any) to satisfy the Rule interface. The data is read back typed in
// IsValid via GetPreparedAs[T].
func (c *ConditionSideEffect[T]) Prepare(ctx context.Context) (any, error) {
	if c.prepare == nil {
		return nil, nil
	}
	data, err := c.prepare(ctx)
	if err != nil {
		return nil, err
	}
	recordPrepared(ctx, c, data)
	return data, nil
}

func (c *ConditionSideEffect[T]) Name() string {
	return c.name
}

// IsValid evaluates the condition using the typed data that Prepare retrieved
// (read back via GetPreparedAs[T], keyed by this condition instance).
func (c *ConditionSideEffect[T]) IsValid(ctx context.Context) bool {
	data, ok := GetPreparedAs[T](ctx, c)
	if !ok {
		return false
	}
	return c.condition(ctx, data)
}

func (c *ConditionSideEffect[T]) IsPure() bool {
	return false
}

// NewConditionSideEffect creates a condition with side effects (e.g., fetching
// data). T is the type of the data loaded during Prepare. The prepare function
// returns a value of type T; the condition function receives it typed, so
// neither side touches `any`.
//
// Prepare returns the retrieved data and records it in the per-evaluation
// preparedStore keyed by this condition; IsValid reads it back typed. The
// condition keeps no state, so a tree built with it can be reused and shared
// across goroutines.
//
// Example:
//
//	cond := rules.NewConditionSideEffect[User](
//	    "userActive",
//	    func(ctx context.Context) (User, error) {
//	        return db.GetUser(ctx, userID)
//	    },
//	    func(ctx context.Context, user User) bool {
//	        return user.Active
//	    },
//	)
func NewConditionSideEffect[T any](
	name string,
	prepare func(ctx context.Context) (T, error),
	condition func(ctx context.Context, data T) bool,
) Condition {
	if prepare == nil {
		prepare = func(context.Context) (T, error) { var zero T; return zero, nil }
	}
	if condition == nil {
		condition = func(context.Context, T) bool { return false }
	}
	return &ConditionSideEffect[T]{
		name:      name,
		prepare:   prepare,
		condition: condition,
	}
}

// TypedRulePure is a pure rule that reads its input data of type T from the
// data registry at validation time. Prepare is a no-op. Used by NewTypedRule.
type TypedRulePure[T any] struct {
	RuleBase
	name string
	fn   func(ctx context.Context, data T) error
}

var _ Rule = (*TypedRulePure[any])(nil)

// Name returns the rule name.
func (r *TypedRulePure[T]) Name() string { return r.name }

// Prepare implements Rule interface. It's a no-op for pure rules.
func (r *TypedRulePure[T]) Prepare(context.Context) (any, error) { return nil, nil }

// Validate reads the typed data from the context registry via GetAs[T] and
// calls the wrapped function. Returns a TYPE_MISMATCH error when the registered
// data is not of type T.
func (r *TypedRulePure[T]) Validate(ctx context.Context) error {
	data, ok := GetAs[T](ctx)
	if !ok {
		var zero T
		raw, _ := Get(ctx)
		return Error{
			Field: r.name,
			Err:   fmt.Sprintf("expected data of type %T, got %T", zero, raw),
			Code:  "TYPE_MISMATCH",
		}
	}
	return r.fn(ctx, data)
}

// NewTypedRule creates a type-safe rule that reads data of type T from the
// data registry at validation time. Returns a TYPE_MISMATCH error if the
// registered data is not of type T.
//
// This is the primary way to create reusable validation rules that work with
// the data registry pattern; the closure receives data of type T directly,
// never `any`.
//
// Example:
//
//	rule := rules.NewTypedRule[User]("checkAge", func(ctx context.Context, user User) error {
//	    if user.Age < 18 {
//	        return fmt.Errorf("must be 18 or older")
//	    }
//	    return nil
//	})
func NewTypedRule[T any](name string, fn func(ctx context.Context, data T) error) Rule {
	return &TypedRulePure[T]{name: name, fn: fn}
}

// NewTypedCondition creates a type-safe condition that reads data of type T
// from the data registry and returns false when the registered data is not of
// type T. It is intended for pure conditions (no side effects).
//
// Example:
//
//	condition := rules.NewTypedCondition[User]("isAdult", func(ctx context.Context, user User) bool {
//	    return user.Age >= 18
//	})
func NewTypedCondition[T any](name string, fn func(ctx context.Context, data T) bool) Condition {
	return &ConditionFunc{
		name: name,
		predicate: func(ctx context.Context) bool {
			data, ok := GetAs[T](ctx)
			if !ok {
				return false
			}
			return fn(ctx, data)
		},
		pure: true,
	}
}

// TypedRuleDataFunc is a rule with Prepare support and type-safe data access.
// In is the input type read from the data registry; T is the loaded data type
// retrieved during Prepare. The rule keeps no state: Prepare records its
// retrieved data in the per-evaluation preparedStore keyed by this rule, and
// Validate reads it back typed via GetPreparedAs[T]. A single tree built with
// it can be reused and shared across goroutines.
type TypedRuleDataFunc[In any, T any] struct {
	RuleBase
	name    string
	prepare func(ctx context.Context, input In) (T, error)
	fn      func(ctx context.Context, input In, data T) error
}

var _ Rule = (*TypedRuleDataFunc[any, any])(nil)

// Name returns the rule name.
func (r *TypedRuleDataFunc[In, T]) Name() string {
	return r.name
}

// Prepare reads the typed input from the data registry, runs the prepare
// function, and records the retrieved data in the per-evaluation preparedStore
// keyed by this rule. The rule keeps no state.
func (r *TypedRuleDataFunc[In, T]) Prepare(ctx context.Context) (any, error) {
	input, ok := GetAs[In](ctx)
	if !ok {
		var zero In
		return nil, Error{
			Field: r.name,
			Err:   fmt.Sprintf("expected input of type %T, got different type", zero),
			Code:  "TYPE_MISMATCH",
		}
	}

	if r.prepare == nil {
		// No prepare step: record the zero value of T so Validate can still
		// read it back with GetPreparedAs[T].
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
// (of type T) recorded by Prepare, and runs the validation function. Both
// reads are typed — no `any` conversion at the rule level.
func (r *TypedRuleDataFunc[In, T]) Validate(ctx context.Context) error {
	input, ok := GetAs[In](ctx)
	if !ok {
		var zero In
		return Error{
			Field: r.name,
			Err:   fmt.Sprintf("expected input of type %T, got different type", zero),
			Code:  "TYPE_MISMATCH",
		}
	}

	loaded, ok := GetPreparedAs[T](ctx, r)
	if !ok {
		return Error{
			Field: r.name,
			Err:   "validation data from prepare not available",
			Code:  "DATA_NOT_PREPARED",
		}
	}

	return r.fn(ctx, input, loaded)
}

// NewTypedRuleWithPrepare creates a type-safe rule with Prepare support.
// This is useful when you need to fetch additional data or perform side effects
// before validation (e.g., checking a database, calling an API).
//
// In is the input type read from the data registry; T is the loaded data type.
// Prepare returns the retrieved data and records it in the per-evaluation
// preparedStore keyed by this rule; the validate function reads it back typed
// (via GetPreparedAs[T]) as its third argument. The rule keeps no state, so a
// tree built with it can be reused and shared across goroutines.
//
// Example:
//
//	rule := rules.NewTypedRuleWithPrepare[User, Permissions](
//	    "checkEmailUniqueness",
//	    func(ctx context.Context, user User) (Permissions, error) {
//	        return db.CheckEmailExists(ctx, user.Email)
//	    },
//	    func(ctx context.Context, user User, perms Permissions) error {
//	        if perms.Exists {
//	            return fmt.Errorf("email already in use")
//	        }
//	        return nil
//	    },
//	)
func NewTypedRuleWithPrepare[In any, T any](
	name string,
	prepare func(ctx context.Context, input In) (T, error),
	validate func(ctx context.Context, input In, data T) error,
) Rule {
	if validate == nil {
		validate = func(ctx context.Context, input In, data T) error { return nil }
	}
	return &TypedRuleDataFunc[In, T]{
		name:    name,
		prepare: prepare,
		fn:      validate,
	}
}

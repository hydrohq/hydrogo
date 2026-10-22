package collections

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	// ErrRuleViolated is returned when a save or delete breaks a rule; the
	// text after ": " is the rule's message.
	ErrRuleViolated = errors.New("rule violated")
	// ErrNoStates is returned for a state change on a type without states.
	ErrNoStates = errors.New("collection has no states")
	// ErrTransitionNotAllowed is returned for a state change the machine does
	// not allow from the record's current state or while its condition fails.
	ErrTransitionNotAllowed = errors.New("transition not allowed")
	// ErrTransitionForbidden is returned when the caller lacks the
	// transition's permission.
	ErrTransitionForbidden = errors.New("transition forbidden")
)

const (
	maxRules       = 20
	maxTransitions = 50
	maxMessage     = 200
)

// Rule is a validation or immutability rule on a type. Exactly one form:
//
//	{assert: "quantity > 0", message: "..."}           every saved record must satisfy it
//	{when: "status == 'posted'", assert: "total > 0"}  ... only while when holds for the saved record
//	{when: "status == 'posted'", readonly: true}       a stored record for which when holds cannot be edited or deleted
//
// Expressions follow Expr. A readonly rule looks at the stored record, an
// assert at the record about to be saved (computed fields and the state
// already filled in). State changes (Service.Transition) are not blocked by
// readonly rules, so "posted" records can still be voided through a transition.
type Rule struct {
	When     string `json:"when,omitempty"`
	Assert   string `json:"assert,omitempty"`
	Readonly bool   `json:"readonly,omitempty"`
	Message  string `json:"message,omitempty"`
}

// StateMachine restricts how a select field can change. New records start in
// Initial; the field cannot be edited directly, only moved along Transitions
// through Service.Transition.
type StateMachine struct {
	Field       string       `json:"field"`
	Initial     string       `json:"initial"`
	Transitions []Transition `json:"transitions"`
}

// Transition allows moving from one state to another, optionally only while
// When holds for the record and only for callers with Permission.
type Transition struct {
	From       string `json:"from"`
	To         string `json:"to"`
	When       string `json:"when,omitempty"`
	Permission string `json:"permission,omitempty"`
	Message    string `json:"message,omitempty"`
}

var permissionName = regexp.MustCompile(`^[a-z][a-z0-9_.]{0,63}$`)

func (s Schema) fieldIDs() map[string]bool {
	ids := make(map[string]bool, len(s.Fields))
	for _, field := range s.Fields {
		ids[field.ID] = true
	}
	return ids
}

// condition parses an expression that may only read fields of the schema.
func (s Schema) condition(source string) error {
	expr, err := ParseExpr(source)
	if err != nil {
		return err
	}
	ids := s.fieldIDs()
	for _, name := range expr.Fields() {
		if !ids[name] {
			return fmt.Errorf("%w: unknown field %q", ErrInvalidExpression, name)
		}
	}
	return nil
}

func (s Schema) validateRulesAndStates() error {
	if len(s.Rules) > maxRules {
		return ErrInvalidSchema
	}
	for _, rule := range s.Rules {
		if len(rule.Message) > maxMessage {
			return ErrInvalidSchema
		}
		switch {
		case rule.Readonly && (rule.Assert != "" || rule.When == ""):
			return ErrInvalidSchema
		case !rule.Readonly && rule.Assert == "":
			return ErrInvalidSchema
		}
		for _, source := range []string{rule.When, rule.Assert} {
			if source != "" && s.condition(source) != nil {
				return ErrInvalidSchema
			}
		}
	}
	if s.States == nil {
		return nil
	}
	machine := s.States
	var field *Field
	for i := range s.Fields {
		if s.Fields[i].ID == machine.Field {
			field = &s.Fields[i]
		}
	}
	if field == nil || field.Kind != Select || field.Required || field.Unique || field.Localized || field.Sequence != nil || field.Computed != "" {
		return ErrInvalidSchema
	}
	if !contains(field.Options, machine.Initial) || len(machine.Transitions) == 0 || len(machine.Transitions) > maxTransitions {
		return ErrInvalidSchema
	}
	seen := map[[2]string]bool{}
	for _, t := range machine.Transitions {
		key := [2]string{t.From, t.To}
		if !contains(field.Options, t.From) || !contains(field.Options, t.To) || t.From == t.To || seen[key] || len(t.Message) > maxMessage {
			return ErrInvalidSchema
		}
		seen[key] = true
		if t.Permission != "" && !permissionName.MatchString(t.Permission) {
			return ErrInvalidSchema
		}
		if t.When != "" && s.condition(t.When) != nil {
			return ErrInvalidSchema
		}
	}
	return nil
}

// holds evaluates a condition against a record's data.
func holds(source string, data map[string]any) (bool, error) {
	expr, err := ParseExpr(source)
	if err != nil {
		return false, err
	}
	return expr.EvalBool(func(name string) Value { return valueOf(data[name]) })
}

func violation(rule Rule, fallback string) error {
	message := strings.TrimSpace(rule.Message)
	if message == "" {
		message = fallback
	}
	return fmt.Errorf("%w: %s", ErrRuleViolated, message)
}

// checkAsserts runs the assert rules on the record about to be saved.
func (s Schema) checkAsserts(data map[string]any) error {
	for _, rule := range s.Rules {
		if rule.Assert == "" {
			continue
		}
		if rule.When != "" {
			applies, err := holds(rule.When, data)
			if err != nil {
				return violation(rule, "rule could not be evaluated")
			}
			if !applies {
				continue
			}
		}
		ok, err := holds(rule.Assert, data)
		if err != nil || !ok {
			return violation(rule, "the record breaks a rule: "+rule.Assert)
		}
	}
	return nil
}

// checkReadonly fails when a readonly rule locks the stored record.
func (s Schema) checkReadonly(stored map[string]any) error {
	for _, rule := range s.Rules {
		if !rule.Readonly {
			continue
		}
		locked, err := holds(rule.When, stored)
		if err != nil || locked {
			return violation(rule, "the record is locked and cannot be changed")
		}
	}
	return nil
}

func (s Schema) hasReadonlyRules() bool {
	for _, rule := range s.Rules {
		if rule.Readonly {
			return true
		}
	}
	return false
}

package validators

import (
	"context"
	"testing"
)

func TestStepValueValidator(t *testing.T) {
	testCases := []struct {
		name    string
		value   float64
		step    float64
		offset  float64
		wantErr bool
	}{
		// Valid cases
		{
			name:    "integer multiple",
			value:   10,
			step:    5,
			offset:  0,
			wantErr: false,
		},
		{
			name:    "float multiple",
			value:   1.0,
			step:    0.5,
			offset:  0,
			wantErr: false,
		},
		{
			name:    "integer multiple with offset",
			value:   12,
			step:    5,
			offset:  2,
			wantErr: false,
		},
		{
			name:    "float multiple with offset",
			value:   1.2,
			step:    0.5,
			offset:  0.2,
			wantErr: false,
		},
		{
			name:    "zero value",
			value:   0,
			step:    5,
			offset:  0,
			wantErr: false,
		},
		{
			name:    "negative value multiple",
			value:   -10,
			step:    5,
			offset:  0,
			wantErr: false,
		},
		{
			name:    "negative value multiple with offset",
			value:   -8,
			step:    5,
			offset:  2,
			wantErr: false,
		},
		{
			name:    "close to multiple (within tolerance)",
			value:   10.0000000001,
			step:    5,
			offset:  0,
			wantErr: false,
		},

		// Invalid cases
		{
			name:    "integer not a multiple",
			value:   11,
			step:    5,
			offset:  0,
			wantErr: true,
		},
		{
			name:    "float not a multiple",
			value:   1.1,
			step:    0.5,
			offset:  0,
			wantErr: true,
		},
		{
			name:    "integer not a multiple with offset",
			value:   13,
			step:    5,
			offset:  2,
			wantErr: true,
		},
		{
			name:    "float not a multiple with offset",
			value:   1.3,
			step:    0.5,
			offset:  0.2,
			wantErr: true,
		},
		{
			name:    "zero step",
			value:   10,
			step:    0,
			offset:  0,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := StepValueValidator(tc.value, tc.step, tc.offset)
			if (err != nil) != tc.wantErr {
				t.Errorf("StepValueValidator() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestNewStepValueValidator(t *testing.T) {
	t.Run("valid rule", func(t *testing.T) {
		rule := NewStepValueValidator(func() int { return 10 }, 5, 0)
		if err := rule.Validate(context.Background()); err != nil {
			t.Errorf("NewStepValueValidator() validation failed, error = %v", err)
		}
	})

	t.Run("invalid rule", func(t *testing.T) {
		rule := NewStepValueValidator(func() float64 { return 10.1 }, 5.0, 0.0)
		if err := rule.Validate(context.Background()); err == nil {
			t.Errorf("NewStepValueValidator() validation should have failed, but it didn't")
		}
	})

	t.Run("rule with zero step", func(t *testing.T) {
		rule := NewStepValueValidator(func() int { return 10 }, 0, 0)
		if err := rule.Validate(context.Background()); err == nil {
			t.Errorf("NewStepValueValidator() with zero step should have failed, but it didn't")
		}
	})

	t.Run("rule name", func(t *testing.T) {
		rule := NewStepValueValidator(func() int { return 10 }, 5, 0)
		if rule.Name() != "step_value_validator" {
			t.Errorf("NewStepValueValidator() rule name is incorrect, got %s, want %s", rule.Name(), "step_value_validator")
		}
	})
}

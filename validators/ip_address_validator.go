package validators

import (
	"fmt"
	"net"

	"github.com/mishudark/rules"
)

// validateIPv4Address is a rule that validates if a string is a valid IPv4 address.
func validateIPv4Address(value string) error {
	ip := net.ParseIP(value)
	if ip == nil || ip.To4() == nil {
		return fmt.Errorf("'%s' is not a valid IPv4 address", value)
	}
	return nil
}

// NewValidateIPv4Address returns a new rule that validates if a string is a valid IPv4 address.
func NewValidateIPv4Address(value string) rules.Rule {
	return rules.NewRulePure("validate_ipv4_address", func() error {
		return validateIPv4Address(value)
	})
}

// validateIPv6Address is a rule that validates if a string is a valid IPv6 address.
func validateIPv6Address(value string) error {
	ip := net.ParseIP(value)
	if ip == nil || ip.To4() != nil {
		return fmt.Errorf("'%s' is not a valid IPv6 address", value)
	}
	return nil
}

// NewValidateIPv6Address returns a new rule that validates if a string is a valid IPv6 address.
func NewValidateIPv6Address(value string) rules.Rule {
	return rules.NewRulePure("validate_ipv6_address", func() error {
		return validateIPv6Address(value)
	})
}

// validateIPv46Address is a rule that validates if a string is a valid IPv4 or IPv6 address.
func validateIPv46Address(value string) error {
	if net.ParseIP(value) == nil {
		return fmt.Errorf("'%s' is not a valid IPv4 or IPv6 address", value)
	}
	return nil
}

// NewValidateIPv46Address returns a new rule that validates if a string is a valid IPv4 or IPv6 address.
func NewValidateIPv46Address(value string) rules.Rule {
	return rules.NewRulePure("validate_ipv46_address", func() error {
		return validateIPv46Address(value)
	})
}

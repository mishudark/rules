package validators

import (
	"testing"
)

func TestValidateIPv4Address(t *testing.T) {
	testCases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "valid_ipv4",
			value:   "192.168.1.1",
			wantErr: false,
		},
		{
			name:    "invalid_ipv4",
			value:   "not-an-ip",
			wantErr: true,
		},
		{
			name:    "ipv6_address",
			value:   "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			wantErr: true,
		},
		{
			name:    "empty_string",
			value:   "",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateIPv4Address(tc.value)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateIPv4Address() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestValidateIPv6Address(t *testing.T) {
	testCases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "valid_ipv6",
			value:   "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			wantErr: false,
		},
		{
			name:    "invalid_ipv6",
			value:   "not-an-ip",
			wantErr: true,
		},
		{
			name:    "ipv4_address",
			value:   "192.168.1.1",
			wantErr: true,
		},
		{
			name:    "empty_string",
			value:   "",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateIPv6Address(tc.value)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateIPv6Address() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestValidateIPv46Address(t *testing.T) {
	testCases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "valid_ipv4",
			value:   "192.168.1.1",
			wantErr: false,
		},
		{
			name:    "valid_ipv6",
			value:   "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			wantErr: false,
		},
		{
			name:    "invalid_ip",
			value:   "not-an-ip",
			wantErr: true,
		},
		{
			name:    "empty_string",
			value:   "",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateIPv46Address(tc.value)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateIPv46Address() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

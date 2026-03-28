package utils

import (
	"testing"
)

func TestValidateSubdomain(t *testing.T) {
	tests := []struct {
		name      string
		subdomain string
		wantErr   bool
	}{
		{
			name:      "valid subdomain",
			subdomain: "test",
			wantErr:   false,
		},
		{
			name:      "subdomain with dash",
			subdomain: "test-subdomain",
			wantErr:   false,
		},
		{
			name:      "subdomain with underscore",
			subdomain: "test_subdomain",
			wantErr:   false,
		},
		{
			name:      "subdomain with uppercase letters",
			subdomain: "TestSubdomain",
			wantErr:   false,
		},
		{
			name:      "subdomain with leading dash",
			subdomain: "-test",
			wantErr:   true,
		},
		{
			name:      "subdomain with trailing dash",
			subdomain: "test-",
			wantErr:   true,
		},
		{
			name:      "subdomain with leading underscore",
			subdomain: "_test",
			wantErr:   true,
		},
		{
			name:      "subdomain with trailing underscore",
			subdomain: "test_",
			wantErr:   true,
		},
		{
			name:      "subdomain with special characters",
			subdomain: "test@subdomain",
			wantErr:   true,
		},
		{
			name:      "subdomain with dot",
			subdomain: "test.subdomain",
			wantErr:   true,
		},
		{
			name:      "subdomain with multiple dots",
			subdomain: "test.subdomain.com",
			wantErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateSubdomain(test.subdomain)
			if (err != nil) != test.wantErr {
				t.Errorf("ValidateSubdomain(%q) = %v, wantErr %v", test.subdomain, err, test.wantErr)
			}
		})
	}
}

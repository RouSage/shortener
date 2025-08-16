package appvalidator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateShortCode(t *testing.T) {
	type shortCode struct {
		Code string `validate:"shortcode"`
	}

	tests := []struct {
		name     string
		value    shortCode
		expected bool
	}{
		{name: "valid alpha", value: shortCode{Code: "shortcode"}, expected: true},
		{name: "valid numeric", value: shortCode{Code: "1234567890"}, expected: true},
		{name: "valid alpha numeric", value: shortCode{Code: "aBC123"}, expected: true},
		{name: "valid shortcode", value: shortCode{Code: "short_CODE-123"}, expected: true},
		{name: "invalid shortcode", value: shortCode{Code: "short$%"}, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validate := New()
			err := validate.Validate(tt.value)

			errors := validate.FormatErrors(err)
			if tt.expected {
				assert.NoError(t, err)
				assert.Len(t, errors, 0, "got no errors")
			} else {
				assert.Error(t, err)
				assert.Len(t, errors, 1, "got more than one error")
				assert.Equal(t, "Short code cannot contain special characters", errors[0].Message, "wrong error message")
				assert.Equal(t, "code", errors[0].Field, "wrong error field")
			}
		})
	}
}

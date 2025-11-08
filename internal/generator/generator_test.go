package generator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShortUrl(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		length         int
		expectedLength int
	}{
		{name: "uses default length", length: 0, expectedLength: defaultLength},
		{name: "uses default length for negative", length: -1, expectedLength: defaultLength},
		{name: "uses custom length", length: 10, expectedLength: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shortUrl, err := ShortUrl(ctx, tt.length)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedLength, len(shortUrl))
		})
	}
}

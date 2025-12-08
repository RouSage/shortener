package generator

import (
	"context"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const defaultLength = 8

var tracer = otel.Tracer("github.com/rousage/shortener/internal/generator")

func ShortUrl(ctx context.Context, length int) (string, error) {
	_, span := tracer.Start(ctx, "generator.ShortUrl")
	defer span.End()

	if length <= 0 {
		span.AddEvent("invalid length, using default", trace.WithAttributes(attribute.Int("length", length)), trace.WithAttributes(attribute.Int("default", defaultLength)))
		length = defaultLength
	}

	id, err := gonanoid.New(length)
	if err != nil {
		span.SetStatus(codes.Error, "nanoid generation failed")
		span.RecordError(err)
		return "", err
	}

	return id, nil
}

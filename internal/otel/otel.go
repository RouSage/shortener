package otel

import (
	"context"
	"errors"

	"github.com/rousage/shortener/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

var ServiceName = semconv.ServiceNameKey.String("shortener")

func SetupOTelSDK(ctx context.Context, cfg config.Otel) (func(context.Context) error, error) {
	var shutdownFuncs []func(context.Context) error
	var err error

	// shutdown calls cleanup functions registered via shutdownFuncs
	// The errors from the calls are joined
	// Each registered cleanup will be invoked once
	shutdown := func(ctx context.Context) error {
		var err error
		for _, f := range shutdownFuncs {
			err = errors.Join(err, f(ctx))
		}
		shutdownFuncs = nil

		return err
	}

	// handleErr calls shutdown for cleanup and makes sure that all errors are returned
	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	res, err := resource.New(ctx, resource.WithAttributes(ServiceName))
	if err != nil {
		handleErr(err)
		return shutdown, err
	}

	tracerProvider, err := newTracerProvider(ctx, res, cfg)
	if err != nil {
		handleErr(err)
		return shutdown, err
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	return shutdown, err
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newTracerProvider(ctx context.Context, res *resource.Resource, cfg config.Otel) (*sdktrace.TracerProvider, error) {
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpointURL(cfg.TracesEndpoint))
	if err != nil {
		return nil, err
	}

	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SamplingRatio))

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(traceExporter),
	)

	return tracerProvider, nil
}

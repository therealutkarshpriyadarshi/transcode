package tracing

import (
	"context"
	"fmt"
	"io"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

// InitTracer initializes the Jaeger tracer
func InitTracer(serviceName, jaegerEndpoint string) (opentracing.Tracer, io.Closer, error) {
	cfg := &config.Configuration{
		ServiceName: serviceName,
		Sampler: &config.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1, // Sample all traces in production, adjust as needed
		},
		Reporter: &config.ReporterConfig{
			LogSpans:           false,
			CollectorEndpoint:  jaegerEndpoint,
			BufferFlushInterval: 1,
		},
	}

	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize tracer: %w", err)
	}

	opentracing.SetGlobalTracer(tracer)
	return tracer, closer, nil
}

// StartSpan starts a new span with the given operation name
func StartSpan(ctx context.Context, operationName string) (opentracing.Span, context.Context) {
	span, ctx := opentracing.StartSpanFromContext(ctx, operationName)
	return span, ctx
}

// FinishSpan finishes a span
func FinishSpan(span opentracing.Span) {
	if span != nil {
		span.Finish()
	}
}

// LogError logs an error to the span
func LogError(span opentracing.Span, err error) {
	if span != nil && err != nil {
		span.SetTag("error", true)
		span.LogKV("error", err.Error())
	}
}

// SetTag sets a tag on the span
func SetTag(span opentracing.Span, key string, value interface{}) {
	if span != nil {
		span.SetTag(key, value)
	}
}

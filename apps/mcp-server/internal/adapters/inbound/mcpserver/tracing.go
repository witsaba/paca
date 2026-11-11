package mcpserver

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// The names of the attributes the spans of the tools carry.
//
// The names of the protocol are the ones of the conventions of the Model
// Context Protocol: they are still in development, and they are the names
// an operator expects to read in the collector.
const (
	// attributeMethod names the method of the protocol the span belongs to.
	attributeMethod = "mcp.method.name"

	// attributeTool names the tool the span belongs to.
	attributeTool = "mcp.tool.name"

	// attributeErrorType names the type of the error that ended the call.
	attributeErrorType = "error.type"
)

// methodToolCall is the method of the protocol that calls a tool.
//
// It is the only method that leaves a span behind: the others carry no
// work. The handshake of the transport, the list of the tools and the ping
// the agent sends every few minutes say nothing about the work of the
// service, and a span for each of them fills the traces with the noise of
// the protocol.
const methodToolCall = "tools/call"

// failureTool is the type of the failure of a call whose result says it
// failed: the SDK hides the error of the tool behind that flag.
const failureTool = "tool.error"

// tracerName is the name of the tracer of the transport.
const tracerName = "paca/mcp-server"

// tracingMiddleware returns the middleware that traces the calls of the
// tools.
//
// The span of a call is the child of the span of the request that carried
// it: the context of the middleware descends from the context of the
// request, so the trace of a call holds the transport, the call and
// everything the call does.
//
// The span carries the name of the tool and nothing else. The arguments of
// a tool and its result are the data of the people who use the product —
// a name, a phone, a sum — and they never reach the store of the traces.
// The type of the error is recorded instead of its message: the messages of
// the domain carry the values they refuse, and the type says the same
// thing without the value.
func tracingMiddleware(tracer trace.Tracer, logger *slog.Logger) sdk.Middleware {
	return func(next sdk.MethodHandler) sdk.MethodHandler {
		return func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
			if method != methodToolCall {
				return next(ctx, method, req)
			}

			tool, ok := toolName(req)
			if !ok {
				return next(ctx, method, req)
			}

			started := time.Now()
			ctx, span := tracer.Start(ctx, tool,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					attribute.String(attributeMethod, method),
					attribute.String(attributeTool, tool),
				),
			)

			result, err := next(ctx, method, req)
			failure := failureOf(result, err)
			if failure != "" {
				span.SetStatus(codes.Error, failure)
				span.SetAttributes(attribute.String(attributeErrorType, failure))
			}
			span.End()

			logger.InfoContext(ctx, "the tool call ended",
				slog.String("tool", tool),
				slog.String("outcome", outcomeOf(failure)),
				slog.String("error_type", failure),
				slog.Duration("duration", time.Since(started)),
			)

			return result, err
		}
	}
}

// failureOf answers how a call ended, as the type of the failure.
//
// The SDK wraps the error of a tool in the result of the call and answers
// no error: a refusal of the domain therefore arrives here as a result that
// says it failed. The message of the failure is never read: it carries the
// values the domain refuses, and the type says the same thing without the
// value.
//
// It answers the empty string when the call ended well.
func failureOf(result sdk.Result, err error) string {
	if err != nil {
		return fmt.Sprintf("%T", err)
	}

	if call, ok := result.(*sdk.CallToolResult); ok && call.IsError {
		return failureTool
	}

	return ""
}

// toolName answers the name of the tool the request calls.
//
// The parameters of the request arrive in the raw form of the protocol: the
// SDK hands them to the middleware before the tool unmarshals them. The
// typed form is accepted too, so the middleware works with either.
//
// It answers false when the request carries no parameters of a tool: the
// middleware never guesses, and a request it cannot name leaves no span.
func toolName(req sdk.Request) (string, bool) {
	switch params := req.GetParams().(type) {
	case *sdk.CallToolParams:
		if params == nil || params.Name == "" {
			return "", false
		}

		return params.Name, true
	case *sdk.CallToolParamsRaw:
		if params == nil || params.Name == "" {
			return "", false
		}

		return params.Name, true
	default:
		return "", false
	}
}

// outcomeOf answers the outcome of a call, for the line of the log.
func outcomeOf(failure string) string {
	if failure != "" {
		return "error"
	}

	return "ok"
}

// tracerOf answers the tracer of the transport, or the one that does
// nothing.
//
// A transport without a tracer provider is a transport that traces nothing:
// the service works the same, and the calls leave no span.
func tracerOf(provider trace.TracerProvider) trace.Tracer {
	if provider == nil {
		provider = noop.NewTracerProvider()
	}

	return provider.Tracer(tracerName)
}

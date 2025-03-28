package trace

import (
	"context"

	"github.com/google/uuid"
)

// Protocol representation of protocol type.
//
// For example, "kafka", "rabbitmq" or "http"
type Protocol string

// Key representation of key.
//
// For example, "trace_id" or "X-Trace-ID"
type Key string

// Generator function which generate new trace id
type Generator func(ctx context.Context) string

func (key Key) String() string {
	return string(key)
}

func (protocol Protocol) String() string {
	return string(protocol)
}

const defaultKey = "bgo_trace_id"

var ProtocolAny Protocol = "any"

var defaultGenerator Generator = func(ctx context.Context) string {
	return uuid.NewString()
}

// Tracer manipulate trace id by [context.Context]
//
// Has 2 modes:
//
//	Master - generate trace id if it doesn't exist
//	Not-Master - doesn't generate new trace id, only pass already existing trace id
type Tracer struct {
	master     bool
	keys       map[Protocol]Key
	uniqueKeys map[Key]struct{}
	generator  Generator
}

// NewTracer creates [Tracer] instance.
//
// By default, register Protocol "any"
//
// Could be provided default key for Protocol "any"
func NewTracer(key ...string) *Tracer {
	anyKey := defaultKey
	if len(key) > 0 {
		anyKey = key[0]
	}

	return &Tracer{
		keys: map[Protocol]Key{
			ProtocolAny: Key(anyKey),
		},
		uniqueKeys: map[Key]struct{}{
			Key(anyKey): {},
		},
		generator: defaultGenerator,
	}
}

// AmIMaster returns state of master
func (tracer *Tracer) AmIMaster() bool {
	return tracer.master
}

// IAmMaster sets new state of master
func (tracer *Tracer) IAmMaster(master bool) *Tracer {
	tracer.master = master
	return tracer
}

// RegisterProtocol registers new protocol with new key.
//
// If protocol already exist skips setting
func (tracer *Tracer) RegisterProtocol(protocol Protocol, key Key) *Tracer {
	if _, ok := tracer.keys[protocol]; ok {
		return tracer
	}

	tracer.uniqueKeys[key] = struct{}{}
	tracer.keys[protocol] = key
	return tracer
}

// SetGenerator sets new Generator.
//
// By default uses defaultGenerator which generates uuid
func (tracer *Tracer) SetGenerator(generator Generator) *Tracer {
	tracer.generator = generator
	return tracer
}

// Set sets new trace id to provided context.
//
// Sets only if tracer in master mode.
//
// Sets trace id to all protocols
func (tracer *Tracer) Set(ctx context.Context) context.Context {
	if !tracer.master {
		return ctx
	}

	_, ok := tracer.TryGet(ctx)
	if ok {
		return ctx
	}

	traceID := tracer.generator(ctx)
	for key := range tracer.uniqueKeys {
		ctx = context.WithValue(ctx, key.String(), traceID)
	}

	return ctx
}

// TryGet return trace id and state if exists.
//
// Uses all registered protocols
func (tracer *Tracer) TryGet(ctx context.Context) (string, bool) {
	for _, key := range tracer.keys {
		traceID := ctx.Value(key)
		if traceID == nil {
			return "", false
		}

		traceIdString, ok := traceID.(string)
		if !ok {
			return "", false
		}

		return traceIdString, true
	}

	return "", false
}

// Get calls TryGet but state
func (tracer *Tracer) Get(ctx context.Context) string {
	traceID, _ := tracer.TryGet(ctx)
	return traceID
}

// TryGetByProtocol return trace id by provided [Protocol] with state
func (tracer *Tracer) TryGetByProtocol(ctx context.Context, protocol Protocol) (string, bool) {
	key, ok := tracer.keys[protocol]
	if !ok {
		return "", false
	}

	traceID := ctx.Value(key.String())
	if traceID == nil {
		return "", false
	}

	traceIdString, ok := traceID.(string)
	if !ok {
		return "", false
	}

	return traceIdString, true
}

// GetByProtocol calls TryGetByProtocol but state
func (tracer *Tracer) GetByProtocol(ctx context.Context, protocol Protocol) string {
	traceID, _ := tracer.TryGetByProtocol(ctx, protocol)
	return traceID
}

// Exist check if any protocol trace id exists
func (tracer *Tracer) Exist(ctx context.Context) bool {
	_, ok := tracer.TryGet(ctx)
	return ok
}

// ExistProtocol check if trace id by protocol exists
func (tracer *Tracer) ExistProtocol(ctx context.Context) bool {
	_, ok := tracer.TryGetByProtocol(ctx, ProtocolAny)
	return ok
}

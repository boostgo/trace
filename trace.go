// Package trace provides tracing tools.
// Features:
// - Different trace id keys for different protocols.
// - Setting trace id to context.
// - Reading trace id from context.
// - Trace master manipulating.
// - Setting custom trace id generator.
package trace

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/boostgo/collection/mapx"
	"github.com/boostgo/convert"
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

const (
	defaultKey = "trace_id"
)

var (
	ProtocolAny Protocol = "any"

	_master     atomic.Bool
	_keys                 = mapx.NewAsyncMap[Protocol, Key]().RWLocker(&sync.RWMutex{})
	_uniqueKeys           = mapx.NewAsyncMap[Key, struct{}]()
	_generator  Generator = func(ctx context.Context) string {
		return uuid.NewString()
	}
)

func IAmMaster(master bool) {
	_master.Store(master)
}

func AmIMaster() bool {
	return _master.Load()
}

// By default, register Protocol "any"
//
// Could be provided default key for Protocol "any"
func init() {
	_keys.Store(ProtocolAny, defaultKey)
	_uniqueKeys.Store(defaultKey, struct{}{})
}

// RegisterProtocol registers new protocol with new key.
//
// If protocol already exist skips setting
func RegisterProtocol(protocol Protocol, key Key) {
	if _, ok := _keys.Load(protocol); ok {
		return
	}

	_keys.Store(protocol, key)
	_uniqueKeys.Store(key, struct{}{})
}

// SetGenerator sets new Generator.
//
// By default uses default generator which generates uuid
func SetGenerator(generator Generator) {
	_generator = generator
}

// Set sets new trace id to provided context.
//
// Sets only if tracer in master mode.
//
// Sets trace id to all protocols
func Set(ctx context.Context) context.Context {
	if _, ok := TryGet(ctx); ok {
		return ctx
	}

	traceID := _generator(ctx)
	_uniqueKeys.Each(func(key Key, value struct{}) bool {
		ctx = context.WithValue(ctx, key.String(), traceID)
		return true
	})

	return ctx
}

func SetID(ctx context.Context, id string) context.Context {
	_, ok := TryGet(ctx)
	if ok {
		return ctx
	}

	_keys.Each(func(protocol Protocol, key Key) bool {
		ctx = context.WithValue(ctx, key.String(), id)
		return true
	})

	return ctx
}

// TryGet return trace id and state if exists.
//
// Uses all registered protocols
func TryGet(ctx context.Context) (string, bool) {
	var traceID string
	var found bool

	_keys.Each(func(protocol Protocol, key Key) bool {
		tID := ctx.Value(key.String())
		if tID == nil {
			return true
		}

		convertedTraceID := convert.String(tID)
		if convertedTraceID == "" {
			return true
		}

		traceID = convertedTraceID
		found = true
		return false
	})

	return traceID, found
}

// Get calls TryGet but state
func Get(ctx context.Context) string {
	traceID, _ := TryGet(ctx)
	return traceID
}

// TryGetByProtocol return trace id by provided [Protocol] with state
func TryGetByProtocol(ctx context.Context, protocol Protocol) (string, bool) {
	key, ok := _keys.Load(protocol)
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
func GetByProtocol(ctx context.Context, protocol Protocol) string {
	traceID, _ := TryGetByProtocol(ctx, protocol)
	return traceID
}

// Exist check if any protocol trace id exists
func Exist(ctx context.Context) bool {
	_, ok := TryGet(ctx)
	return ok
}

// ExistProtocol check if trace id by protocol exists
func ExistProtocol(ctx context.Context) bool {
	_, ok := TryGetByProtocol(ctx, ProtocolAny)
	return ok
}

// Keys return all registered unique keys
func Keys() []string {
	keys := make([]string, 0, _keys.Len())
	_uniqueKeys.Each(func(key Key, value struct{}) bool {
		keys = append(keys, string(key))
		return true
	})
	return keys
}

// Generate new trace id
func Generate(ctx context.Context) string {
	return _generator(ctx)
}

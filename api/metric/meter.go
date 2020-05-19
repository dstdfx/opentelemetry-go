// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metric

import (
	"context"

	"go.opentelemetry.io/otel/api/kv"
)

// The file is organized as follows:
//
//  - Provider interface
//  - Meter struct
//  - RecordBatch
//  - BatchObserver
//  - Synchronous instrument constructors (2 x int64,float64)
//  - Asynchronous instrument constructors (1 x int64,float64)
//  - Batch asynchronous constructors (1 x int64,float64)
//  - Internals

// Provider supports named Meter instances.
type Provider interface {
	// Meter gets a named Meter interface.  If the name is an
	// empty string, the provider uses a default name.
	Meter(name string) Meter
}

// Meter is the OpenTelemetry metric API, based on a `MeterImpl`
// implementation and the `Meter` library name.
//
// An uninitialized Meter is a no-op implementation.
type Meter struct {
	impl        MeterImpl
	libraryName string
}

// RecordBatch atomically records a batch of measurements.
func (m Meter) RecordBatch(ctx context.Context, ls []kv.KeyValue, ms ...Measurement) {
	if m.impl == nil {
		return
	}
	m.impl.RecordBatch(ctx, ls, ms...)
}

// NewBatchObserver creates a new BatchObserver that supports
// making batches of observations for multiple instruments.
func (m Meter) NewBatchObserver(callback BatchObserverCallback) BatchObserver {
	return BatchObserver{
		meter:  m,
		runner: newBatchAsyncRunner(callback),
	}
}

// NewInt64Counter creates a new integer Counter instrument with the
// given name, customized with options.  May return an error if the
// name is invalid (e.g., empty) or improperly registered (e.g.,
// duplicate registration).
func (m Meter) NewInt64Counter(name string, options ...Option) (Int64Counter, error) {
	return wrapInt64CounterInstrument(
		m.newSync(name, CounterKind, Int64NumberKind, options))
}

// NewFloat64Counter creates a new floating point Counter with the
// given name, customized with options.  May return an error if the
// name is invalid (e.g., empty) or improperly registered (e.g.,
// duplicate registration).
func (m Meter) NewFloat64Counter(name string, options ...Option) (Float64Counter, error) {
	return wrapFloat64CounterInstrument(
		m.newSync(name, CounterKind, Float64NumberKind, options))
}

// NewInt64UpDownCounter creates a new integer UpDownCounter instrument with the
// given name, customized with options.  May return an error if the
// name is invalid (e.g., empty) or improperly registered (e.g.,
// duplicate registration).
func (m Meter) NewInt64UpDownCounter(name string, options ...Option) (Int64UpDownCounter, error) {
	return wrapInt64UpDownCounterInstrument(
		m.newSync(name, UpDownCounterKind, Int64NumberKind, options))
}

// NewFloat64UpDownCounter creates a new floating point UpDownCounter with the
// given name, customized with options.  May return an error if the
// name is invalid (e.g., empty) or improperly registered (e.g.,
// duplicate registration).
func (m Meter) NewFloat64UpDownCounter(name string, options ...Option) (Float64UpDownCounter, error) {
	return wrapFloat64UpDownCounterInstrument(
		m.newSync(name, UpDownCounterKind, Float64NumberKind, options))
}

// NewInt64ValueRecorder creates a new integer ValueRecorder instrument with the
// given name, customized with options.  May return an error if the
// name is invalid (e.g., empty) or improperly registered (e.g.,
// duplicate registration).
func (m Meter) NewInt64ValueRecorder(name string, opts ...Option) (Int64ValueRecorder, error) {
	return wrapInt64ValueRecorderInstrument(
		m.newSync(name, ValueRecorderKind, Int64NumberKind, opts))
}

// NewFloat64ValueRecorder creates a new floating point ValueRecorder with the
// given name, customized with options.  May return an error if the
// name is invalid (e.g., empty) or improperly registered (e.g.,
// duplicate registration).
func (m Meter) NewFloat64ValueRecorder(name string, opts ...Option) (Float64ValueRecorder, error) {
	return wrapFloat64ValueRecorderInstrument(
		m.newSync(name, ValueRecorderKind, Float64NumberKind, opts))
}

// RegisterInt64ValueObserver creates a new integer ValueObserver instrument
// with the given name, running a given callback, and customized with
// options.  May return an error if the name is invalid (e.g., empty)
// or improperly registered (e.g., duplicate registration).
func (m Meter) RegisterInt64ValueObserver(name string, callback Int64ObserverCallback, opts ...Option) (Int64ValueObserver, error) {
	if callback == nil {
		return wrapInt64ValueObserverInstrument(NoopAsync{}, nil)
	}
	return wrapInt64ValueObserverInstrument(
		m.newAsync(name, ValueObserverKind, Int64NumberKind, opts,
			newInt64AsyncRunner(callback)))
}

// RegisterFloat64ValueObserver creates a new floating point ValueObserver with
// the given name, running a given callback, and customized with
// options.  May return an error if the name is invalid (e.g., empty)
// or improperly registered (e.g., duplicate registration).
func (m Meter) RegisterFloat64ValueObserver(name string, callback Float64ObserverCallback, opts ...Option) (Float64ValueObserver, error) {
	if callback == nil {
		return wrapFloat64ValueObserverInstrument(NoopAsync{}, nil)
	}
	return wrapFloat64ValueObserverInstrument(
		m.newAsync(name, ValueObserverKind, Float64NumberKind, opts,
			newFloat64AsyncRunner(callback)))
}

// RegisterInt64ValueObserver creates a new integer ValueObserver instrument
// with the given name, running in a batch callback, and customized with
// options.  May return an error if the name is invalid (e.g., empty)
// or improperly registered (e.g., duplicate registration).
func (b BatchObserver) RegisterInt64ValueObserver(name string, opts ...Option) (Int64ValueObserver, error) {
	if b.runner == nil {
		return wrapInt64ValueObserverInstrument(NoopAsync{}, nil)
	}
	return wrapInt64ValueObserverInstrument(
		b.meter.newAsync(name, ValueObserverKind, Int64NumberKind, opts, b.runner))
}

// RegisterFloat64ValueObserver creates a new floating point ValueObserver with
// the given name, running in a batch callback, and customized with
// options.  May return an error if the name is invalid (e.g., empty)
// or improperly registered (e.g., duplicate registration).
func (b BatchObserver) RegisterFloat64ValueObserver(name string, opts ...Option) (Float64ValueObserver, error) {
	if b.runner == nil {
		return wrapFloat64ValueObserverInstrument(NoopAsync{}, nil)
	}
	return wrapFloat64ValueObserverInstrument(
		b.meter.newAsync(name, ValueObserverKind, Float64NumberKind, opts,
			b.runner))
}

// MeterImpl returns the underlying MeterImpl of this Meter.
func (m Meter) MeterImpl() MeterImpl {
	return m.impl
}

// newAsync constructs one new asynchronous instrument.
func (m Meter) newAsync(
	name string,
	mkind Kind,
	nkind NumberKind,
	opts []Option,
	runner AsyncRunner,
) (
	AsyncImpl,
	error,
) {
	if m.impl == nil {
		return NoopAsync{}, nil
	}
	desc := NewDescriptor(name, mkind, nkind, opts...)
	desc.config.LibraryName = m.libraryName
	return m.impl.NewAsyncInstrument(desc, runner)
}

// newSync constructs one new synchronous instrument.
func (m Meter) newSync(
	name string,
	metricKind Kind,
	numberKind NumberKind,
	opts []Option,
) (
	SyncImpl,
	error,
) {
	if m.impl == nil {
		return NoopSync{}, nil
	}
	desc := NewDescriptor(name, metricKind, numberKind, opts...)
	desc.config.LibraryName = m.libraryName
	return m.impl.NewSyncInstrument(desc)
}
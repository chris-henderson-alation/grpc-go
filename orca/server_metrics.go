/*
 *
 * Copyright 2023 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package orca

import (
	"sync"

	v3orcapb "github.com/cncf/xds/go/xds/data/orca/v3"
)

// ServerMetrics is the data returned from a server to a client to describe the
// current state of the server and/or the cost of a request when used per-call.
type ServerMetrics struct {
	CPUUtilization float64 // CPU utilization: [0, 1.0]; unset=-1
	MemUtilization float64 // Memory utilization: [0, 1.0]; unset=-1
	QPS            float64 // queries per second: [0, inf); unset=-1
	EPS            float64 // errors per second: [0, inf); unset=-1

	// The following maps must never be nil.

	Utilization  map[string]float64 // Custom fields: [0, 1.0]
	RequestCost  map[string]float64 // Custom fields: [0, inf); not sent OOB
	NamedMetrics map[string]float64 // Custom fields: [0, inf); not sent OOB
}

// toLoadReportProto dumps sm as an OrcaLoadReport proto.
func (sm *ServerMetrics) toLoadReportProto() *v3orcapb.OrcaLoadReport {
	ret := &v3orcapb.OrcaLoadReport{
		Utilization:  sm.Utilization,
		RequestCost:  sm.RequestCost,
		NamedMetrics: sm.NamedMetrics,
	}
	if sm.CPUUtilization != -1 {
		ret.CpuUtilization = sm.CPUUtilization
	}
	if sm.MemUtilization != -1 {
		ret.MemUtilization = sm.MemUtilization
	}
	if sm.QPS != -1 {
		ret.RpsFractional = sm.QPS
	}
	if sm.EPS != -1 {
		ret.Eps = sm.EPS
	}
	return ret
}

// merge merges o into sm, overwriting any values present in both.
func (sm *ServerMetrics) merge(o *ServerMetrics) {
	if o.CPUUtilization != -1 {
		sm.CPUUtilization = o.CPUUtilization
	}
	if o.MemUtilization != -1 {
		sm.MemUtilization = o.MemUtilization
	}
	if o.QPS != -1 {
		sm.QPS = o.QPS
	}
	if o.EPS != -1 {
		sm.EPS = o.EPS
	}
	mergeMap(sm.Utilization, o.Utilization)
	mergeMap(sm.RequestCost, o.RequestCost)
	mergeMap(sm.NamedMetrics, o.NamedMetrics)
}

func mergeMap(a, b map[string]float64) {
	for k, v := range b {
		a[k] = v
	}
}

// ServerMetricsRecorder allows for recording and providing out of band server
// metrics.
type ServerMetricsRecorder interface {
	ServerMetricsProvider

	// SetCPUUtilization sets the relevant server metric.
	SetCPUUtilization(float64)
	// DeleteCPUUtilization deletes the relevant server metric to prevent it
	// from being sent.
	DeleteCPUUtilization()

	// SetMemoryUtilization sets the relevant server metric.
	SetMemoryUtilization(float64)
	// DeleteMemoryUtilization deletes the relevant server metric to prevent it
	// from being sent.
	DeleteMemoryUtilization()

	// SetQPS sets the relevant server metric.
	SetQPS(float64)
	// DeleteQPS deletes the relevant server metric to prevent it from being
	// sent.
	DeleteQPS()

	// SetEPS sets the relevant server metric.
	SetEPS(float64)
	// DeleteEPS deletes the relevant server metric to prevent it from being
	// sent.
	DeleteEPS()

	// SetNamedUtilization sets the relevant server metric.
	SetNamedUtilization(name string, val float64)
	// DeleteNamedUtilization deletes the relevant server metric to prevent it
	// from being sent.
	DeleteNamedUtilization(name string)
}

type serverMetricsRecorder struct {
	mu    sync.Mutex     // protects state
	state *ServerMetrics // the current metrics
}

// NewServerMetricsRecorder returns an in-memory store for ServerMetrics and
// allows for safe setting and retrieving of ServerMetrics.  Also implements
// ServerMetricsProvider for use with NewService.
func NewServerMetricsRecorder() ServerMetricsRecorder {
	return newServerMetricsRecorder()
}

func newServerMetricsRecorder() *serverMetricsRecorder {
	return &serverMetricsRecorder{
		state: &ServerMetrics{
			CPUUtilization: -1,
			MemUtilization: -1,
			QPS:            -1,
			EPS:            -1,
			Utilization:    make(map[string]float64),
			RequestCost:    make(map[string]float64),
			NamedMetrics:   make(map[string]float64),
		},
	}
}

// ServerMetrics returns a copy of the current ServerMetrics.
func (s *serverMetricsRecorder) ServerMetrics() *ServerMetrics {
	s.mu.Lock()
	defer s.mu.Unlock()
	return &ServerMetrics{
		CPUUtilization: s.state.CPUUtilization,
		MemUtilization: s.state.MemUtilization,
		QPS:            s.state.QPS,
		EPS:            s.state.EPS,
		Utilization:    copyMap(s.state.Utilization),
		RequestCost:    copyMap(s.state.RequestCost),
		NamedMetrics:   copyMap(s.state.NamedMetrics),
	}
}

func copyMap(m map[string]float64) map[string]float64 {
	ret := make(map[string]float64, len(m))
	for k, v := range m {
		ret[k] = v
	}
	return ret
}

// SetCPUUtilization records a measurement for the CPU utilization metric.
func (s *serverMetricsRecorder) SetCPUUtilization(val float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.CPUUtilization = val
}

// DeleteCPUUtilization deletes the relevant server metric to prevent it from
// being sent.
func (s *serverMetricsRecorder) DeleteCPUUtilization() {
	s.SetCPUUtilization(-1)
}

// SetMemoryUtilization records a measurement for the memory utilization metric.
func (s *serverMetricsRecorder) SetMemoryUtilization(val float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.MemUtilization = val
}

// DeleteMemoryUtilization deletes the relevant server metric to prevent it
// from being sent.
func (s *serverMetricsRecorder) DeleteMemoryUtilization() {
	s.SetMemoryUtilization(-1)
}

// SetQPS records a measurement for the QPS metric.
func (s *serverMetricsRecorder) SetQPS(val float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.QPS = val
}

// DeleteQPS deletes the relevant server metric to prevent it from being sent.
func (s *serverMetricsRecorder) DeleteQPS() {
	s.SetQPS(-1)
}

// SetEPS records a measurement for the EPS metric.
func (s *serverMetricsRecorder) SetEPS(val float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.EPS = val
}

// DeleteEPS deletes the relevant server metric to prevent it from being sent.
func (s *serverMetricsRecorder) DeleteEPS() {
	s.SetEPS(-1)
}

// SetNamedUtilization records a measurement for a utilization metric uniquely
// identifiable by name.
func (s *serverMetricsRecorder) SetNamedUtilization(name string, val float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Utilization[name] = val
}

// DeleteNamedUtilization deletes any previously recorded measurement for a
// utilization metric uniquely identifiable by name.
func (s *serverMetricsRecorder) DeleteNamedUtilization(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.state.Utilization, name)
}

// SetRequestCost records a measurement for a utilization metric uniquely
// identifiable by name.
func (s *serverMetricsRecorder) SetRequestCost(name string, val float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.RequestCost[name] = val
}

// DeleteRequestCost deletes any previously recorded measurement for a
// utilization metric uniquely identifiable by name.
func (s *serverMetricsRecorder) DeleteRequestCost(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.state.RequestCost, name)
}

// SetNamedMetric records a measurement for a utilization metric uniquely
// identifiable by name.
func (s *serverMetricsRecorder) SetNamedMetric(name string, val float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.NamedMetrics[name] = val
}

// DeleteNamedMetric deletes any previously recorded measurement for a
// utilization metric uniquely identifiable by name.
func (s *serverMetricsRecorder) DeleteNamedMetric(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.state.NamedMetrics, name)
}

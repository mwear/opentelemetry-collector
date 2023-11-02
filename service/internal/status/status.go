// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package status // import "go.opentelemetry.io/collector/service/internal/status"

import (
	"errors"
	"fmt"
	"sync"

	"go.opentelemetry.io/collector/component"
)

// onTransitionFunc receives a component.StatusEvent on a successful state transition
type onTransitionFunc func(*component.StatusEvent)

// errInvalidStateTransition is returned for invalid state transitions
var errInvalidStateTransition = errors.New("invalid state transition")

// fsm is a finite state machine that models transitions for component status
type fsm struct {
	current      *component.StatusEvent
	transitions  map[component.Status]map[component.Status]struct{}
	onTransition onTransitionFunc
}

// transition will attempt to execute a state transition. If it's successful, it calls the
// onTransitionFunc with a StatusEvent representing the new state. Returns an error if the arguments
// result in an invalid status, or if the state transition is not valid.
func (m *fsm) transition(ev *component.StatusEvent) error {
	if _, ok := m.transitions[m.current.Status()][ev.Status()]; !ok {
		return fmt.Errorf(
			"cannot transition from %s to %s: %w",
			m.current.Status(),
			ev.Status(),
			errInvalidStateTransition,
		)
	}
	m.current = ev
	m.onTransition(ev)
	return nil
}

// newFSM creates a state machine with all valid transitions for component.Status.
// The initial state is set to component.StatusNone.
func newFSM(onTransition onTransitionFunc) *fsm {
	return &fsm{
		current:      component.NewStatusEvent(component.StatusNone),
		onTransition: onTransition,
		transitions: map[component.Status]map[component.Status]struct{}{
			component.StatusNone: {
				component.StatusStarting: {},
			},
			component.StatusStarting: {
				component.StatusOK:               {},
				component.StatusRecoverableError: {},
				component.StatusPermanentError:   {},
				component.StatusFatalError:       {},
				component.StatusStopping:         {},
			},
			component.StatusOK: {
				component.StatusRecoverableError: {},
				component.StatusPermanentError:   {},
				component.StatusFatalError:       {},
				component.StatusStopping:         {},
			},
			component.StatusRecoverableError: {
				component.StatusOK:             {},
				component.StatusPermanentError: {},
				component.StatusFatalError:     {},
				component.StatusStopping:       {},
			},
			component.StatusPermanentError: {},
			component.StatusFatalError:     {},
			component.StatusStopping: {
				component.StatusRecoverableError: {},
				component.StatusPermanentError:   {},
				component.StatusFatalError:       {},
				component.StatusStopped:          {},
			},
			component.StatusStopped: {},
		},
	}
}

// NotifyStatusFunc is the receiver of status events after successful state transitions
type NotifyStatusFunc func(*component.InstanceID, *component.StatusEvent)

// ServiceStatusFunc is the expected type of ReportComponentStatus for servicetelemetry.Settings
type ServiceStatusFunc func(*component.InstanceID, *component.StatusEvent) error

// errStatusNotReady is returned when trying to report status before service start
var errStatusNotReady = errors.New("report component status is not ready until service start")

// Reporter handles component status reporting
type Reporter struct {
	fsmMap map[*component.InstanceID]*fsm
	fsmMu  sync.RWMutex

	ready   bool
	readyMu sync.RWMutex

	onStatusChange NotifyStatusFunc
}

// NewReporter a reporter that will invoke the NotifyStatusFunc when a component's status
// has changed.
func NewReporter(onStatusChange NotifyStatusFunc) *Reporter {
	return &Reporter{
		fsmMap:         make(map[*component.InstanceID]*fsm),
		fsmMu:          sync.RWMutex{},
		readyMu:        sync.RWMutex{},
		onStatusChange: onStatusChange,
	}
}

// ReportComponentStatus reports status for the given InstanceID
func (r *Reporter) ReportComponentStatus(
	id *component.InstanceID,
	ev *component.StatusEvent,
) error {
	if !r.isReady() {
		return errStatusNotReady
	}
	r.fsmMu.Lock()
	defer r.fsmMu.Unlock()
	return r.componentFSM(id).transition(ev)
}

// ReportComponentStatusIf reports status for the given InstanceID, if cond, a predicate that
// receives the current status as an argument, evaluates to true.
func (r *Reporter) ReportComponentStatusIf(
	id *component.InstanceID,
	ev *component.StatusEvent,
	cond func(component.Status) bool,
) error {
	if !r.isReady() {
		return errStatusNotReady
	}
	r.fsmMu.Lock()
	defer r.fsmMu.Unlock()
	fsm := r.componentFSM(id)
	if cond(fsm.current.Status()) {
		return fsm.transition(ev)
	}
	return nil
}

func (r *Reporter) componentFSM(id *component.InstanceID) *fsm {
	fsm, ok := r.fsmMap[id]
	if !ok {
		fsm = newFSM(func(ev *component.StatusEvent) {
			r.onStatusChange(id, ev)
		})
		r.fsmMap[id] = fsm
	}
	return fsm
}

// Ready enables status reporting
func (r *Reporter) Ready() {
	r.readyMu.Lock()
	defer r.readyMu.Unlock()
	r.ready = true
}

func (r *Reporter) isReady() bool {
	r.readyMu.RLock()
	defer r.readyMu.RUnlock()
	return r.ready
}

// NewComponentStatusFunc returns a function to be used as ReportComponentStatus for
// component.TelemetrySettings, which differs from servicetelemetry.Settings in that
// the component version is tied to specific component instance.
func NewComponentStatusFunc(
	id *component.InstanceID,
	srvStatus ServiceStatusFunc,
) component.StatusFunc {
	return func(ev *component.StatusEvent) error {
		return srvStatus(id, ev)
	}
}

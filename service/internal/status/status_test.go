// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package status

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
)

func TestStatusFSM(t *testing.T) {
	for _, tc := range []struct {
		name               string
		reportedStatuses   []component.Status
		expectedStatuses   []component.Status
		expectedErrorCount int
	}{
		{
			name: "successful startup and shutdown",
			reportedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusOK,
				component.StatusStopping,
				component.StatusStopped,
			},
			expectedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusOK,
				component.StatusStopping,
				component.StatusStopped,
			},
		},
		{
			name: "component recovered",
			reportedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusRecoverableError,
				component.StatusOK,
				component.StatusStopping,
				component.StatusStopped,
			},
			expectedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusRecoverableError,
				component.StatusOK,
				component.StatusStopping,
				component.StatusStopped,
			},
		},
		{
			name: "repeated events are errors",
			reportedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusOK,
				component.StatusRecoverableError,
				component.StatusRecoverableError,
				component.StatusRecoverableError,
				component.StatusOK,
				component.StatusStopping,
				component.StatusStopped,
			},
			expectedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusOK,
				component.StatusRecoverableError,
				component.StatusOK,
				component.StatusStopping,
				component.StatusStopped,
			},
			expectedErrorCount: 2,
		},
		{
			name: "PermanentError is terminal",
			reportedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusOK,
				component.StatusPermanentError,
				component.StatusOK,
			},
			expectedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusOK,
				component.StatusPermanentError,
			},
			expectedErrorCount: 1,
		},
		{
			name: "FatalError is terminal",
			reportedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusOK,
				component.StatusFatalError,
				component.StatusOK,
			},
			expectedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusOK,
				component.StatusFatalError,
			},
			expectedErrorCount: 1,
		},
		{
			name: "Stopped is terminal",
			reportedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusOK,
				component.StatusStopping,
				component.StatusStopped,
				component.StatusOK,
			},
			expectedStatuses: []component.Status{
				component.StatusStarting,
				component.StatusOK,
				component.StatusStopping,
				component.StatusStopped,
			},
			expectedErrorCount: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var receivedStatuses []component.Status
			fsm := newFSM(
				func(ev *component.StatusEvent) {
					receivedStatuses = append(receivedStatuses, ev.Status())
				},
			)

			errorCount := 0
			for _, status := range tc.reportedStatuses {
				if err := fsm.Event(status); err != nil {
					errorCount++
					require.ErrorIs(t, err, errInvalidStateTransition)
				}
			}

			require.Equal(t, tc.expectedErrorCount, errorCount)
			require.Equal(t, tc.expectedStatuses, receivedStatuses)
		})
	}
}

func TestStatusEventError(t *testing.T) {
	fsm := newFSM(func(*component.StatusEvent) {})
	fsm.Event(component.StatusStarting)
	// the combination of StatusOK with an error is invalid
	err := fsm.Event(component.StatusOK, component.WithError(assert.AnError))

	require.Error(t, err)
	require.ErrorIs(t, err, component.ErrStatusEventInvalidArgument)
}

func TestStatusFuncs(t *testing.T) {
	id1 := &component.InstanceID{}
	id2 := &component.InstanceID{}

	actualStatuses := make(map[*component.InstanceID][]component.Status)
	statusFunc := func(id *component.InstanceID, ev *component.StatusEvent) {
		actualStatuses[id] = append(actualStatuses[id], ev.Status())
	}

	statuses1 := []component.Status{
		component.StatusStarting,
		component.StatusOK,
		component.StatusStopping,
		component.StatusStopped,
	}

	statuses2 := []component.Status{
		component.StatusStarting,
		component.StatusOK,
		component.StatusRecoverableError,
		component.StatusOK,
		component.StatusStopping,
		component.StatusStopped,
	}

	expectedStatuses := map[*component.InstanceID][]component.Status{
		id1: statuses1,
		id2: statuses2,
	}

	serviceStatusFn := NewServiceStatusFunc(statusFunc)

	comp1Func := NewComponentStatusFunc(id1, serviceStatusFn)
	comp2Func := NewComponentStatusFunc(id2, serviceStatusFn)

	for _, st := range statuses1 {
		comp1Func(st)
	}

	for _, st := range statuses2 {
		comp2Func(st)
	}

	require.Equal(t, expectedStatuses, actualStatuses)
}

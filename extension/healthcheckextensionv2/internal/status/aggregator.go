// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package status // import "go.opentelemetry.io/collector/extension/healthcheckextensionv2/internal/status"

import (
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
)

var extsID = component.NewID("extensions")
var extsIDMap = map[component.ID]struct{}{extsID: {}}

type Aggregator struct {
	startTimestamp        time.Time
	aggregateStatus       *component.StatusEvent
	aggregateStatusesByID map[component.ID]*component.StatusEvent
	componentStatusesByID map[component.ID]map[*component.InstanceID]*component.StatusEvent
	verbose               bool
	mu                    sync.RWMutex
}

type SerializableStatus struct {
	StartTimestamp *time.Time `json:"start_time,omitempty"`
	*SerializableEvent
	ComponentStatuses map[string]*SerializableStatus `json:"components,omitempty"`
}

type SerializableEvent struct {
	status       component.Status
	StatusString string    `json:"status"`
	Err          error     `json:"error,omitempty"`
	Timestamp    time.Time `json:"status_time"`
}

func (ev *SerializableEvent) Status() component.Status {
	return ev.status
}

func toSerializableEvent(ev *component.StatusEvent) *SerializableEvent {
	return &SerializableEvent{
		status:       ev.Status(),
		StatusString: ev.Status().String(),
		Err:          ev.Err(),
		Timestamp:    ev.Timestamp(),
	}
}

func (a *Aggregator) Current() *SerializableStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()

	s := &SerializableStatus{
		SerializableEvent: toSerializableEvent(a.aggregateStatus),
		ComponentStatuses: make(map[string]*SerializableStatus),
		StartTimestamp:    &a.startTimestamp,
	}

	if !a.verbose {
		return s
	}

	for compID, ev := range a.aggregateStatusesByID {
		key := compID.String()
		if compID != extsID {
			key = "pipeline:" + key
		}
		as := &SerializableStatus{
			SerializableEvent: toSerializableEvent(ev),
			ComponentStatuses: make(map[string]*SerializableStatus),
		}
		s.ComponentStatuses[key] = as
		for instance, ev := range a.componentStatusesByID[compID] {
			key := fmt.Sprintf("%s:%s", instance.Kind, instance.ID)
			as.ComponentStatuses[key] = &SerializableStatus{
				SerializableEvent: toSerializableEvent(ev),
			}
		}
	}

	return s
}

func (a *Aggregator) StatusChanged(source *component.InstanceID, event *component.StatusEvent) {
	compIDs := source.PipelineIDs
	// extensions are treated as a pseudo-pipeline
	if source.Kind == component.KindExtension {
		compIDs = extsIDMap
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	for compID := range compIDs {
		var compStatuses map[*component.InstanceID]*component.StatusEvent
		compStatuses, ok := a.componentStatusesByID[compID]
		if !ok {
			compStatuses = make(map[*component.InstanceID]*component.StatusEvent)
		}
		compStatuses[source] = event
		a.componentStatusesByID[compID] = compStatuses
		a.aggregateStatusesByID[compID] = component.AggregateStatusEvent(compStatuses)
	}

	a.aggregateStatus = component.AggregateStatusEvent(a.aggregateStatusesByID)
}

func NewAggregator() *Aggregator {
	return &Aggregator{
		startTimestamp:        time.Now(),
		aggregateStatus:       &component.StatusEvent{},
		aggregateStatusesByID: make(map[component.ID]*component.StatusEvent),
		componentStatusesByID: make(map[component.ID]map[*component.InstanceID]*component.StatusEvent),
		verbose:               true,
		mu:                    sync.RWMutex{},
	}
}

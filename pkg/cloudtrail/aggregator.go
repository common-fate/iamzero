package cloudtrail

import (
	"github.com/common-fate/iamzero/pkg/recommendations"
	"github.com/pkg/errors"
)

// Aggregator reads and deduplicates CloudTrailEvents.
// It calls event.TryConvertToEvent() to convert the CloudTrailEvent
// into our standard AWSEvent format.
// It stores events with a map of uint64 hashes to the event
type Aggregator struct {
	events map[uint64]recommendations.AWSEvent
}

func NewAggregator() Aggregator {
	return Aggregator{
		events: make(map[uint64]recommendations.AWSEvent),
	}
}

// Read a new CloudTrail event, convert it and deduplicate it.
func (a *Aggregator) Read(e CloudTrailLogEntry) error {
	event, err := e.TryConvertToEvent()
	// if the log entry couldn't be mapped we get an ErrNoMapping
	// we simply ignore these errors
	if err != nil && err != ErrNoMapping {
		return errors.Wrap(err, "converting event")
	}

	// if we don't get an event, return early.
	if event == nil {
		return nil
	}

	key, err := recommendations.HashEvent(*event)
	if err != nil {
		return errors.Wrap(err, "hashing event")
	}

	a.events[key] = *event
	return nil
}

// GetEvents returns all events read by the Aggregator
func (a *Aggregator) GetEvents() []recommendations.AWSEvent {
	e := []recommendations.AWSEvent{}

	for _, v := range a.events {
		e = append(e, v)
	}

	return e
}

package storage

import (
	"github.com/common-fate/iamzero/pkg/recommendations"
)

// NoOpEventStorage meets the EventStorage interface but doesn't
// do anything. It allows us to continue to use the `iamzero local` CLI
// command workflows without a hard requirement that Postgres exists.
// In this case, some detail views will just return no information
// (when querying the events related to a specific finding), for example.
// In future we can replace this with an alternative storage driver to be used for
// local CLI usage.
type NoOpEventStorage struct {
}

func (s *NoOpEventStorage) ListForFinding(findingID string) ([]recommendations.AWSEvent, error) {
	return []recommendations.AWSEvent{}, nil
}

func (s *NoOpEventStorage) Create(e recommendations.AWSEvent) error {
	return nil
}

func (s *NoOpEventStorage) Get(id string) (*recommendations.AWSEvent, error) {
	return nil, nil
}

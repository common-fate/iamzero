package storage

import "github.com/common-fate/iamzero/pkg/recommendations"

type EventStorage interface {
	ListForFinding(findingID string) ([]recommendations.AWSEvent, error)
	Get(id string) (*recommendations.AWSEvent, error)
	Create(recommendations.AWSEvent) error
}

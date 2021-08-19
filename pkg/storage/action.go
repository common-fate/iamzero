package storage

import "github.com/common-fate/iamzero/pkg/recommendations"

type ActionStorage interface {
	Add(action recommendations.AWSAction) error
	List() ([]recommendations.AWSAction, error)
	Get(id string) (*recommendations.AWSAction, error)
	ListForPolicy(policyID string) ([]recommendations.AWSAction, error)
	SetStatus(id string, status string) error
	Update(action recommendations.AWSAction) error
}

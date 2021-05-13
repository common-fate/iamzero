package storage

import (
	"errors"

	"github.com/common-fate/iamzero/pkg/recommendations"
)

type AlertStorage struct {
	alerts []recommendations.AWSAlert
}

func NewAlertStorage() AlertStorage {
	return AlertStorage{alerts: []recommendations.AWSAlert{}}
}

func (a *AlertStorage) Add(alert recommendations.AWSAlert) {
	a.alerts = append(a.alerts, alert)
}

func (a *AlertStorage) List() []recommendations.AWSAlert {
	return a.alerts
}

func (a *AlertStorage) Get(id string) *recommendations.AWSAlert {
	for _, alert := range a.alerts {
		if alert.ID == id {
			return &alert
		}
	}
	return nil
}

func (a *AlertStorage) SetStatus(id string, status string) error {
	for i, alert := range a.alerts {
		if alert.ID == id {
			a.alerts[i].Status = status
			return nil
		}
	}
	return errors.New("could not find alert")
}

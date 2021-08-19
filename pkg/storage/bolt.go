package storage

import (
	"os"
	"path"

	"github.com/asdine/storm/v3"
	"github.com/common-fate/iamzero/pkg/recommendations"
)

func OpenBoltDB() (*storm.DB, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	folder := path.Join(home, ".iamzero")
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		err := os.Mkdir(folder, os.FileMode(0700))
		if err != nil {
			return nil, err
		}
	}
	file := path.Join(folder, "findings.db")

	db, err := storm.Open(file)
	if err != nil {
		return nil, err
	}

	err = db.Init(recommendations.Policy{})
	if err != nil {
		return nil, err
	}
	err = db.Init(recommendations.AWSAction{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

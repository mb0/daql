package main

import (
	"os"

	"github.com/mb0/daql/mig"
)

func db() (string, error) {
	db := *dbFlag
	if db == "" {
		db = os.Getenv("DAQL_DB")
	}
	return db, nil
}

func dataset(fname string) (mig.Dataset, error) {
	return nil, nil
}

func dump(args []string) error {
	c, err := db()
	if err != nil {
		return err
	}
	_ = c
	return nil
}

func backup(args []string) error {
	c, err := db()
	if err != nil {
		return err
	}
	_ = c
	return nil
}

func replay(args []string) error {
	ds, err := dataset(args[0])
	if err != nil {
		return err
	}
	c, err := db()
	if err != nil {
		return err
	}
	_, _ = ds, c
	return nil
}

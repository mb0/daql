package main

import "github.com/mb0/daql/mig"

func history(pr *Project) (*mig.History, error) {
	return nil, nil
}

func status(args []string) error {
	pr, err := project()
	if err != nil {
		return err
	}
	history(pr)
	return nil
}

func record(args []string) error {
	pr, err := project()
	if err != nil {
		return err
	}
	history(pr)
	return nil
}

func migrate(args []string) error {
	pr, err := project()
	if err != nil {
		return err
	}
	history(pr)
	dataset(args[0])
	return nil
}

func scrub(args []string) error {
	return nil
}

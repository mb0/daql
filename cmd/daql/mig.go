package main

import (
	"fmt"
	"strings"

	"github.com/mb0/daql/mig"
)

func status(args []string) error {
	pr, err := project()
	if err != nil {
		return err
	}
	curr := pr.First()
	last := pr.Last().First()
	var vers string
	if last.Vers == 0 {
		vers = fmt.Sprintf("v%d (unrecorded)", curr.Vers)
	} else if curr.Vers != last.Vers {
		vers = fmt.Sprintf("v%d (last recorded v%d %s)", curr.Vers, last.Vers,
			last.Date.Format("2006-02-01 15:04"))
	} else {
		vers = fmt.Sprintf("v%d (unchanged, recorded %s)",
			last.Vers, last.Date.Format("2006-02-01 15:04"))
	}
	fmt.Printf("Project: %s %s\n\n", pr.Name, vers)
	fmt.Printf("Models:\n")
	for _, s := range pr.Schemas {
		for _, m := range s.Models {
			mv, _ := pr.Get(m.Qualified())
			chg := "+"
			fmt.Printf("       %s %s v%d\n", chg, mv.Name, mv.Vers)
		}
	}
	fmt.Println()
	return nil
}

func record(args []string) error {
	pr, err := project()
	if err != nil {
		return err
	}
	err = pr.Commit(strings.Join(args, "_"))
	if err == mig.ErrNoChanges {
		fmt.Printf("%s v%d unchanged\n", pr.Name, pr.First().Vers)
		return nil
	}
	if err != nil {
		return err
	}
	fmt.Printf("%s v%d recorded\n", pr.Name, pr.First().Vers)
	return nil
}

func migrate(args []string) error {
	pr, err := project()
	if err != nil {
		return err
	}
	dataset(args[0])
	_ = pr
	return nil
}

func scrub(args []string) error {
	pr, err := project()
	if err != nil {
		return err
	}
	_ = pr
	return nil
}

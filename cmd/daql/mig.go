package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mb0/daql/mig"
)

func status(args []string) error {
	pr, err := project()
	if err != nil {
		return err
	}
	cv := pr.First()
	last := pr.Last()
	lv := last.First()
	var vers string
	if lv.Vers == 0 {
		vers = fmt.Sprintf("v%d (unrecorded)", cv.Vers)
	} else if cv.Vers != lv.Vers || cv.Name != lv.Name {
		vers = fmt.Sprintf("v%d (last recorded v%d %s)", cv.Vers, lv.Vers,
			lv.Date.Format("2006-02-01 15:04"))
	} else {
		vers = fmt.Sprintf("v%d (unchanged, recorded %s)",
			lv.Vers, lv.Date.Format("2006-02-01 15:04"))
	}
	fmt.Printf("Project: %s %s\n", pr.Name, vers)
	changes := pr.Diff(pr.Last())
	chg(changes, cv.Name)
	fmt.Printf("Definition:\n")
	const nodefmt = "       %c %s v%d\n"
	for _, s := range pr.Schemas {
		v, _ := pr.Get(s.Qualified())
		fmt.Printf(nodefmt, chg(changes, v.Name), v.Name, v.Vers)
		for _, m := range s.Models {
			v, _ = pr.Get(m.Qualified())
			fmt.Printf(nodefmt, chg(changes, v.Name), v.Name, v.Vers)
		}
	}
	fmt.Println()
	if chg(changes, lv.Name) != ' ' {
		fmt.Printf("Project renamed from %s to %s\n\n", lv.Name, cv.Name)
	}
	if len(changes) > 0 {
		fmt.Printf("Deletions:\n")
		dels := make([]string, 0, len(changes))
		for k := range changes {
			dels = append(dels, k)
		}
		sort.Strings(dels)
		for _, s := range dels {
			v, _ := pr.Get(s)
			fmt.Printf(nodefmt, '-', s, v.Vers)
		}
		fmt.Println()
	}
	return nil
}

func chg(cm map[string]byte, name string) byte {
	if b, ok := cm[name]; ok {
		delete(cm, name)
		return b
	}
	return ' '
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

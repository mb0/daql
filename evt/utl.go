package evt

import (
	"github.com/mb0/xelf/cor"
	"github.com/mb0/xelf/utl"
)

type ByRev []*Event

func (s ByRev) Len() int           { return len(s) }
func (s ByRev) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s ByRev) Less(i, j int) bool { return s[j].Rev.After(s[i].Rev) }

type ByID []*Event

func (s ByID) Len() int           { return len(s) }
func (s ByID) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s ByID) Less(i, j int) bool { return s[i].ID < s[j].ID }

func Collect(evs []*Event, s Sig) (res []*Event) {
	for _, ev := range evs {
		if ev.Sig == s {
			res = append(res, ev)
		}
	}
	return res
}

func CollectAll(evs []*Event) map[Sig][]*Event {
	res := make(map[Sig][]*Event)
	for _, ev := range evs {
		res[ev.Sig] = append(res[ev.Sig], ev)
	}
	return res
}

func Merge(a, b Action) (_ Action, err error) {
	if a.Sig != b.Sig {
		return a, cor.Errorf("event signature mismatch %v != %v", a.Sig, b.Sig)
	}
	switch a.Cmd {
	case "-":
		switch b.Cmd {
		case "+":
			// TODO zero optional arg fields
			return Action{Sig: a.Sig, Cmd: "*", Arg: b.Arg}, nil
		case "*":
			return a, cor.Errorf("modify after delete action for %v", a.Sig)
		case "-":
			return a, cor.Errorf("double delete action for %v", a.Sig)
		}
	case "+", "*":
		switch b.Cmd {
		case "+":
			return a, cor.Errorf("create action for existing %v", a.Sig)
		case "*":
			if a.Cmd == "+" {
				err = utl.ApplyDelta(a.Arg, b.Arg)
			} else {
				err = utl.MergeDeltas(a.Arg, b.Arg)
			}
			return a, err
		case "-":
			return b, nil
		}
	default:
		return a, cor.Errorf("unresolved action %s", a.Cmd)
	}
	return a, cor.Errorf("unresolved action %s", b.Cmd)
}

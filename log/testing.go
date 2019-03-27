package log

type TB interface {
	Errorf(string, ...interface{})
	Fatalf(string, ...interface{})
	Logf(string, ...interface{})
	Helper()
}

type Testing struct {
	TB
	Default
}

func (l *Testing) Debug(m string, s ...interface{}) {
	l.Helper()
	l.Logf(tfmt("DEB ", m, s, l.Tags))
}
func (l *Testing) Error(m string, s ...interface{}) {
	l.Helper()
	l.Errorf(tfmt("ERR ", m, s, l.Tags))
}
func (l *Testing) Crit(m string, s ...interface{}) {
	l.Helper()
	l.Fatalf(tfmt("CRI", m, s, l.Tags))
}
func (l *Testing) With(tags ...interface{}) Logger {
	return &Testing{l.TB, *l.Default.with(tags)}
}

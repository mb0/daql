package log

import (
	"fmt"
	"log"
	"strings"
)

var Root Logger = &Default{}

// Logger is logger interface. The variadic arguments are key value pairs. The key must be a
// string and the value should have a meaningful string representations.
type Logger interface {
	Debug(string, ...interface{})
	Error(string, ...interface{})
	Crit(string, ...interface{})
	With(...interface{}) Logger
}

type Default struct {
	Tags []interface{}
}

func (l *Default) Debug(m string, s ...interface{}) { log.Printf(tfmt("DEB ", m, s, l.Tags)) }
func (l *Default) Error(m string, s ...interface{}) { log.Printf(tfmt("ERR ", m, s, l.Tags)) }
func (l *Default) Crit(m string, s ...interface{})  { log.Printf(tfmt("CRI ", m, s, l.Tags)) }
func (l *Default) With(tags ...interface{}) Logger {
	return l.with(tags)
}
func (l *Default) with(tags ...interface{}) *Default {
	t := make([]interface{}, 0, len(tags)+len(l.Tags))
	t = append(t, tags...)
	t = append(t, l.Tags...)
	return &Default{Tags: t}
}

func tfmt(lvl, msg string, all ...[]interface{}) string {
	var b strings.Builder
	b.WriteString(lvl)
	b.WriteString(msg)
	for _, tags := range all {
		for i, v := range tags {
			if i%2 == 0 {
				b.WriteByte(' ')
			} else {
				b.WriteByte('=')
			}
			b.WriteString(fmt.Sprint(v))
		}
	}
	return b.String()
}

package hub

import "github.com/mb0/xelf/cor"

// A service is a common interface for the last message processor in line.
// It usually is used by wrappers, that handles request parsing and delegate.
type Service interface {
	// Serve handles the message and returns the response data or nil.
	Serve(*Msg) interface{}
}

// Services is a map if message subjects to service processors.
type Services map[string]Service

// Handle calls the service with m's subject or returns an error.
// If the service returns data and c is not nil, a reply is sent to the sender.
func (s Services) Handle(m *Msg, c Conn) error {
	f := s[m.Subj]
	if f == nil {
		return cor.Errorf("service not supported %s", m.Subj)
	}
	res := f.Serve(m)
	if res != nil && c != nil {
		m.From.Chan() <- &Msg{From: c, Subj: m.Subj, Data: res}
	}
	return nil
}

package notify

import "context"

type MockSender struct {
	infoCalled     bool
	criticalCalled bool
	Err            error
	msg            *Message
}

func (s *MockSender) Info(_ context.Context, msg *Message) error {
	s.infoCalled = true
	s.msg = msg
	return s.Err
}

func (s *MockSender) Critical(_ context.Context, msg *Message) error {
	s.criticalCalled = true
	s.msg = msg
	return s.Err
}

func (s *MockSender) InfoWasCalled() bool {
	return s.infoCalled
}

func (s *MockSender) CriticalWasCalled() bool {
	return s.criticalCalled
}

func (s *MockSender) CapturedMessage() *Message {
	return s.msg
}

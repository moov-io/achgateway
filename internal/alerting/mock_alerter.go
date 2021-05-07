package alerting

type MockAlerter struct {
}

func (mn *MockAlerter) AlertError(e error) error {
	return nil
}

package files

import "context"

type MockRepository struct {
	Err error
}

func NewMockRepository() Repository {
	return &MockRepository{}
}

func (r *MockRepository) Record(_ context.Context, file AcceptedFile) error {
	return r.Err
}

func (r *MockRepository) Cancel(_ context.Context, fileID string) error {
	return r.Err
}

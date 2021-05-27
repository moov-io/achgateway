package incoming

import (
	"testing"

	"github.com/moov-io/achgateway/internal/consul"
	"github.com/moov-io/base/log"
)

func TestAcquireLock(t *testing.T) {
	type args struct {
		logger log.Logger
		client *consul.Client
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := AcquireLock(tt.args.logger, tt.args.client); (err != nil) != tt.wantErr {
				t.Errorf("AcquireLock() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

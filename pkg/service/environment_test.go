// generated-from:f3f35f4002746aa851a730b299373b812f8173fc53182b8fb17f63e8fd427fdd DO NOT REMOVE, DO UPDATE

package service_test

import (
	"testing"

	"github.com/moov-io/ach-conductor/pkg/service"
	"github.com/moov-io/ach-conductor/pkg/test"
	"github.com/moov-io/base/log"

	"github.com/stretchr/testify/assert"
)

func Test_Environment_Startup(t *testing.T) {
	a := assert.New(t)

	env := &service.Environment{
		Logger: log.NewDefaultLogger(),
		Config: &service.Config{
			Database: test.TestDatabaseConfig(),
		},
	}

	env, err := service.NewEnvironment(env)
	a.Nil(err)

	t.Cleanup(env.Shutdown)
}

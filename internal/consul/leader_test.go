package consul

import (
	"github.com/moov-io/base/log"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAcquireLock(t *testing.T) {
	a := assert.New(t)
	logger := log.NewDefaultLogger()

	consulClient, err := NewConsulClient(logger, &Config{
		Address:                    "127.0.0.1:8500",
		Scheme:                     "http",
		SessionPath:                "achgateway/test/",
		Tags:                       []string{"test1"},
		HealthCheckIntervalSeconds: 10,
	})
	a.Nil(err)

	testShard := "test"
	consulSessions := map[string]*Session{}

	newSession, err := NewSession(logger, *consulClient, testShard)
	a.Nil(err)
	consulSessions[testShard] = newSession
	a.IsType(&Session{}, consulSessions[testShard])

	err = AcquireLock(logger, consulClient, consulSessions[testShard])
	a.Nil(err)
}

func TestAcquireLockSessionExists(t *testing.T) {
	a := assert.New(t)
	logger := log.NewDefaultLogger()

	consulClient, err := NewConsulClient(logger, &Config{
		Address:                    "127.0.0.1:8500",
		Scheme:                     "http",
		SessionPath:                "achgateway/test/",
		Tags:                       []string{"test1"},
		HealthCheckIntervalSeconds: 10,
	})
	a.Nil(err)

	testShard := "test2"
	consulSessions := map[string]*Session{}

	newSession, err := NewSession(logger, *consulClient, testShard)
	a.Nil(err)
	consulSessions[testShard] = newSession

	if _, exists := consulSessions[testShard]; exists {
		a.IsType(&Session{}, consulSessions[testShard])
	}

	err = AcquireLock(logger, consulClient, consulSessions[testShard])
	a.Nil(err)
}

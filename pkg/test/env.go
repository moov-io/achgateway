// generated-from:976ed7cc55901dca480471d0f0b8254555334a68999c86a72b6a334287376895 DO NOT REMOVE, DO UPDATE

package test

import (
	"testing"

	"github.com/gorilla/mux"
	"github.com/moov-io/base/log"
	"github.com/moov-io/base/stime"
	"github.com/moovfinancial/go-zero-trust/pkg/middleware"
	"github.com/moovfinancial/go-zero-trust/pkg/middleware/middlewaretest"
	"github.com/stretchr/testify/require"

	"github.com/moov-io/ach-conductor/pkg/service"
)

type TestEnvironment struct {
	T          *testing.T
	Assert     *require.Assertions
	StaticTime stime.StaticTimeService
	Claims     middleware.TrustedClaims

	service.Environment
}

func NewEnvironment(t *testing.T, router *mux.Router) *TestEnvironment {
	testEnv := &TestEnvironment{}

	testEnv.T = t
	testEnv.PublicRouter = router
	testEnv.Assert = require.New(t)
	testEnv.Logger = log.NewDefaultLogger()
	testEnv.StaticTime = stime.NewStaticTimeService()
	testEnv.TimeService = testEnv.StaticTime
	testEnv.Claims = middlewaretest.NewRandomClaims()

	cfg, err := service.LoadConfig(testEnv.Logger)
	if err != nil {
		t.Fatal(err)
	}
	testEnv.Config = cfg

	cfg.Database = CreateTestDatabase(t, TestDatabaseConfig())

	claimsFunc := func() middleware.TrustedClaims { return testEnv.Claims }
	mw := middlewaretest.NewTestMiddlewareLazy(testEnv.StaticTime, claimsFunc, "ach-conductor")
	testEnv.ZeroTrustMiddleware = mw.Handler

	_, err = service.NewEnvironment(&testEnv.Environment)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(testEnv.Shutdown)

	return testEnv
}

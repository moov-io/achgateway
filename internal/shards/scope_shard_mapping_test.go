// generated-from:513a6cae9eb5750a973eaf1731a9148f2a8b3404be8120214049077cff8b8d3e DO NOT REMOVE, DO UPDATE

package shards_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/moov-io/achgateway/internal"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/internal/shards"
	"github.com/moov-io/base/log"
	"github.com/moov-io/base/stime"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

type TestEnvironment struct {
	T            *testing.T
	Assert       *require.Assertions
	StaticTime   stime.StaticTimeService
	PublicRouter *mux.Router
	Logger       log.Logger
	TimeService  stime.StaticTimeService
	Config       *service.Config
	Environment  internal.Environment
	Client       *http.Client
}

func NewTestEnvironment(t *testing.T, router *mux.Router) *TestEnvironment {
	testEnv := &TestEnvironment{}

	testEnv.T = t
	testEnv.PublicRouter = router
	testEnv.Assert = require.New(t)
	testEnv.Logger = log.NewDefaultLogger()
	testEnv.StaticTime = stime.NewStaticTimeService()
	testEnv.TimeService = testEnv.StaticTime

	cfg, err := internal.LoadConfig(testEnv.Logger)
	if err != nil {
		t.Fatal(err)
	}
	testEnv.Config = cfg

	_, err = internal.NewEnvironment(&testEnv.Environment)
	if err != nil {
		t.Fatal(err)
	}

	return testEnv
}

type ShardMappingTestScope struct {
	*TestEnvironment

	Repository shards.Repository
	Service    shards.ShardMappingService
	API        *http.Client
}

func ShardMappingTestSetup(t *testing.T) ShardMappingTestScope {
	router := mux.NewRouter()
	testEnv := NewTestEnvironment(t, router)

	repository := shards.NewInMemoryRepository()
	service, err := shards.NewShardMappingService(testEnv.TimeService, testEnv.Logger, repository)
	if err != nil {
		t.Error(err)
	}

	controller := shards.NewShardMappingController(testEnv.Logger, service)
	controller.AppendRoutes(router)

	testAPI := &http.Client{}

	return ShardMappingTestScope{
		TestEnvironment: testEnv,
		Repository:      repository,
		Service:         service,
		API:             testAPI,
	}
}

func (s TestEnvironment) MakeRequest(method string, target string, body interface{}) *http.Request {
	jsonBody := bytes.Buffer{}
	if body != nil {
		json.NewEncoder(&jsonBody).Encode(body)
	}

	return httptest.NewRequest(method, target, &jsonBody)
}

func (s TestEnvironment) MakeCall(req *http.Request, body interface{}) *http.Response {
	rec := httptest.NewRecorder()
	s.PublicRouter.ServeHTTP(rec, req)
	res := rec.Result()
	defer res.Body.Close()

	if body != nil {
		json.NewDecoder(res.Body).Decode(&body)
	}

	return res
}

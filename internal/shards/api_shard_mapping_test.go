// generated-from:ca92f6dd217b1cf0e29194f72c8e104eba4b95a8e8adf572b67add9ab4856e37 DO NOT REMOVE, DO UPDATE

package shards_test

import (
	"github.com/moov-io/achgateway/internal/service"
	"net/http"
	"testing"
)

func Test_ShardMapping_CreateAPI(t *testing.T) {
	s := ShardMappingTestSetup(t)
	create := &service.ShardMapping{
		ShardKey:  "test",
		ShardName: "tester",
	}

	_, resp := clientShardMappingCreate(s, create)
	s.Assert.NotNil(resp)
	defer resp.Body.Close()
	s.Assert.Equal(201, resp.StatusCode)
}

func clientShardMappingCreate(s ShardMappingTestScope, create *service.ShardMapping) (*service.ShardMapping, *http.Response) {
	i := &service.ShardMapping{}
	resp := s.MakeCall(s.MakeRequest("POST", "/shard_mappings", create), i)
	return i, resp
}

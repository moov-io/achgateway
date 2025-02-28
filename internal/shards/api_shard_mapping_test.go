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
	defer resp.Body.Close()
	s.Assert.NotNil(resp)
	s.Assert.Equal(201, resp.StatusCode)
}

func Test_ShardMapping_GetAPI(t *testing.T) {
	s := ShardMappingTestSetup(t)
	create := &service.ShardMapping{
		ShardKey:  "test",
		ShardName: "tester",
	}

	_, resp := clientShardMappingCreate(s, create)
	defer resp.Body.Close()
	s.Assert.NotNil(resp)
	s.Assert.Equal(201, resp.StatusCode)

	shardMapping, resp1 := clientShardMappingGet(s, create.ShardKey)
	defer resp1.Body.Close()
	s.Assert.Equal(200, resp1.StatusCode)

	s.Assert.Equal(create.ShardName, shardMapping.ShardName)
}

func Test_ShardMapping_ListAPI(t *testing.T) {
	s := ShardMappingTestSetup(t)
	create1 := &service.ShardMapping{
		ShardKey:  "test1",
		ShardName: "tester1",
	}
	create2 := &service.ShardMapping{
		ShardKey:  "test2",
		ShardName: "tester2",
	}

	_, resp1 := clientShardMappingCreate(s, create1)
	defer resp1.Body.Close()
	s.Assert.NotNil(resp1)
	_, resp2 := clientShardMappingCreate(s, create2)
	defer resp2.Body.Close()
	s.Assert.NotNil(resp2)

	shardMappings, resp := clientShardMappingList(s)
	defer resp.Body.Close()
	s.Assert.Equal(200, resp.StatusCode)

	s.Assert.Equal(2, len(shardMappings))
}

func clientShardMappingCreate(s ShardMappingTestScope, create *service.ShardMapping) (*service.ShardMapping, *http.Response) {
	i := &service.ShardMapping{}
	resp := s.MakeCall(s.MakeRequest("POST", "/shard_mappings", create), i)
	return i, resp
}

func clientShardMappingGet(s ShardMappingTestScope, shardKey string) (*service.ShardMapping, *http.Response) {
	i := &service.ShardMapping{}
	resp := s.MakeCall(s.MakeRequest("GET", "/shard_mappings/"+shardKey, nil), i)
	return i, resp
}

func clientShardMappingList(s ShardMappingTestScope) ([]service.ShardMapping, *http.Response) {
	var i []service.ShardMapping
	request := s.MakeRequest("GET", "/shard_mappings", nil)
	resp := s.MakeCall(request, &i)
	return i, resp
}

package shards_test

import (
	"github.com/google/uuid"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFacilitatorService_Create(t *testing.T) {
	s := ShardMappingTestSetup(t)

	shardKey := "test"
	shardName := "tester"
	create := &service.ShardMapping{
		ShardKey:  shardKey,
		ShardName: shardName,
	}

	shard, err := s.Service.Create(create)
	require.NoError(t, err)
	require.Equal(t, shardName, shard.ShardName)
}

func TestFacilitatorService_List(t *testing.T) {
	s := ShardMappingTestSetup(t)

	create1 := &service.ShardMapping{
		ShardKey:  "test1",
		ShardName: "tester1",
	}
	create2 := &service.ShardMapping{
		ShardKey:  "test2",
		ShardName: "tester2",
	}
	create3 := &service.ShardMapping{
		ShardKey:  "test3",
		ShardName: "tester3",
	}

	_, err := s.Service.Create(create1)
	require.NoError(t, err)

	_, err = s.Service.Create(create2)
	require.NoError(t, err)

	_, err = s.Service.Create(create3)
	require.NoError(t, err)

	list, err := s.Service.List()
	require.NoError(t, err)
	require.Len(t, list, 3)
}

func TestFacilitatorService_Get(t *testing.T) {
	s := ShardMappingTestSetup(t)

	shardKey := uuid.NewString()
	shardName := "someName"

	create := &service.ShardMapping{
		ShardKey:  shardKey,
		ShardName: shardName,
	}

	_, err := s.Service.Create(create)
	require.NoError(t, err)

	foundName, err := s.Service.Lookup(shardKey)
	require.NoError(t, err)
	require.Equal(t, shardName, foundName)
}

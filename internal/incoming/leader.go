package incoming

import (
	"github.com/hashicorp/consul/api"

	"github.com/moov-io/achgateway/internal/consul"
	"github.com/moov-io/base/log"
)

func AcquireLock(logger log.Logger, client *consul.Client) error  {
	isLeader, _, err := client.ConsulClient.KV().Acquire(&api.KVPair{
		Key:     client.Cfg.SessionName,
		Value:   []byte(client.SessionId),
		Session: client.SessionId,
	}, nil)

	if err != nil {
		return err
	}

	if isLeader {
		return nil
	}
	return logger.Info().LogErrorf("%s is not the leader", client.NodeId).Err()
}

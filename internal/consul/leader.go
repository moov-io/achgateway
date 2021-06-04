package consul

import (
	"github.com/hashicorp/consul/api"

	"github.com/moov-io/base/log"
)

func AcquireLock(logger log.Logger, client *Client, consulSession *Session) error {
	isLeader, _, err := client.ConsulClient.KV().Acquire(&api.KVPair{
		Key:     consulSession.Name,
		Value:   []byte(consulSession.ID),
		Session: consulSession.ID,
	}, nil)

	if err != nil {
		return err
	}

	if isLeader {
		return nil
	}
	return logger.Info().LogErrorf("%s is not the leader", client.NodeId).Err()
}

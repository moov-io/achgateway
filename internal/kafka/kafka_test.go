package kafka_test

import (
	"fmt"
	"testing"

	"github.com/moov-io/achgateway/internal/kafka"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"

	"github.com/stretchr/testify/require"
)

func TestSaramaDebugLogging(t *testing.T) {
	t.Setenv("SARAMA_DEBUG_LOGGING", "yes")

	buf, logger := log.NewBufferLogger()
	kafka.EnableSaramaDebugLogging(logger)

	cfg := service.KafkaConfig{
		Brokers: []string{"127.0.0.1:55555"},
	}

	topic, err := kafka.OpenTopic(log.NewNopLogger(), &cfg)
	require.Nil(t, topic)
	require.Error(t, err)

	fmt.Printf("\n%s\n", buf.String())
	require.Contains(t, buf.String(), `msg="ClientID is the default of 'sarama',`)
}

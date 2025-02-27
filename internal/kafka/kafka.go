package kafka

import (
	"time"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"

	"github.com/Shopify/sarama"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/kafkapubsub"
)

var (
	minKafkaVersion = sarama.V2_6_0_0
)

func OpenTopic(logger log.Logger, cfg *service.KafkaConfig) (*pubsub.Topic, error) {
	config := kafkapubsub.MinimalConfig()
	config.Version = minKafkaVersion
	config.Net.TLS.Enable = cfg.TLS

	config.Net.SASL.Enable = cfg.Key != ""
	config.Net.SASL.Mechanism = sarama.SASLMechanism(cfg.SASLMechanism)

	// Default to PLAIN if no SASL mechanism is specified
	switch cfg.SASLMechanism {
	case "SCRAM-SHA-512":
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
			return &XDGSCRAMClient{HashGeneratorFcn: SHA512}
		}
		config.Net.SASL.Mechanism = sarama.SASLMechanism(cfg.SASLMechanism)

	case "SCRAM-SHA-256":
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
			return &XDGSCRAMClient{HashGeneratorFcn: SHA256}
		}
		config.Net.SASL.Mechanism = sarama.SASLMechanism(cfg.SASLMechanism)

	default:
		config.Net.SASL.Mechanism = sarama.SASLMechanism("PLAIN")
	}

	config.Net.SASL.User = cfg.Key
	config.Net.SASL.Password = cfg.Secret

	if cfg.Producer.MaxMessageBytes > 0 {
		config.Producer.MaxMessageBytes = cfg.Producer.MaxMessageBytes
	}

	logger.Info().
		Set("tls", log.Bool(cfg.TLS)).
		Set("group", log.String(cfg.Group)).
		Set("sasl.enable", log.Bool(config.Net.SASL.Enable)).
		Set("sasl.user", log.String(cfg.Key)).
		Set("topic", log.String(cfg.Topic)).
		Log("opening kafka topic")

	return kafkapubsub.OpenTopic(cfg.Brokers, config, cfg.Topic, nil)
}

func OpenSubscription(logger log.Logger, cfg *service.KafkaConfig) (*pubsub.Subscription, error) {
	config := kafkapubsub.MinimalConfig()
	config.Version = minKafkaVersion
	config.Net.TLS.Enable = cfg.TLS

	config.Net.SASL.Enable = cfg.Key != ""
	// Default to PLAIN if no SASL mechanism is specified
	switch cfg.SASLMechanism {
	case "SCRAM-SHA-512":
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
			return &XDGSCRAMClient{HashGeneratorFcn: SHA512}
		}
		config.Net.SASL.Mechanism = sarama.SASLMechanism(cfg.SASLMechanism)

	case "SCRAM-SHA-256":
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
			return &XDGSCRAMClient{HashGeneratorFcn: SHA256}
		}
		config.Net.SASL.Mechanism = sarama.SASLMechanism(cfg.SASLMechanism)

	default:
		config.Net.SASL.Mechanism = sarama.SASLMechanism("PLAIN")
	}

	config.Net.SASL.User = cfg.Key
	config.Net.SASL.Password = cfg.Secret

	// AutoCommit in Sarama refers to "automated publishing of consumer offsets
	// to the broker" rather than a Kafka broker's meaning of "commit consumer
	// offsets on read" which leads to "at-most-once" delivery.
	config.Consumer.Offsets.AutoCommit.Enable = cfg.AutoCommit

	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Consumer.IsolationLevel = sarama.ReadCommitted

	logger.Info().
		Set("tls", log.Bool(cfg.TLS)).
		Set("group", log.String(cfg.Group)).
		Set("sasl.enable", log.Bool(config.Net.SASL.Enable)).
		Set("sasl.user", log.String(cfg.Key)).
		Set("topic", log.String(cfg.Topic)).
		Log("setting up kafka subscription")

	return kafkapubsub.OpenSubscription(cfg.Brokers, config, cfg.Group, []string{cfg.Topic}, &kafkapubsub.SubscriptionOptions{
		WaitForJoin: 10 * time.Second,
	})
}

package kafka

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"

	"github.com/Shopify/sarama"
	"github.com/aws/aws-msk-iam-sasl-signer-go/signer"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/kafkapubsub"
)

var (
	minKafkaVersion = sarama.V2_6_0_0
)

type MSKAccessTokenProvider struct {
	Region      string
	Profile     string
	RoleARN     string
	SessionName string
}

func (m *MSKAccessTokenProvider) Token() (*sarama.AccessToken, error) {
	var token string
	var err error

	// Choose the correct AWS authentication method
	switch {
	case m.Profile != "":
		token, _, err = signer.GenerateAuthTokenFromProfile(context.TODO(), m.Region, m.Profile)
	case m.RoleARN != "":
		token, _, err = signer.GenerateAuthTokenFromRole(context.TODO(), m.Region, m.RoleARN, m.SessionName)
	default:
		token, _, err = signer.GenerateAuthToken(context.TODO(), m.Region)
	}

	if err != nil {
		return nil, err
	}
	return &sarama.AccessToken{Token: token}, nil
}

func OpenTopic(logger log.Logger, cfg *service.KafkaConfig) (*pubsub.Topic, error) {
	config := kafkapubsub.MinimalConfig()
	config.Version = minKafkaVersion
	config.Net.TLS.Enable = cfg.TLS

	config.Net.SASL.Enable = cfg.Key != ""

	switch cfg.SASLMechanism {
	case "AWS_MSK_IAM":
		if cfg.Provider != nil && cfg.Provider.AWS != nil {
			config.Net.SASL.TokenProvider = &MSKAccessTokenProvider{
				Region:      cfg.Provider.AWS.Region,
				Profile:     cfg.Provider.AWS.Profile,
				RoleARN:     cfg.Provider.AWS.RoleARN,
				SessionName: cfg.Provider.AWS.SessionName,
			}
		}
		config.Net.SASL.Mechanism = sarama.SASLTypeOAuth
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = &tls.Config{}

	default:
		config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
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
		Set("sasl.mechanism", log.String(string(config.Net.SASL.Mechanism))).
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
	case "AWS_MSK_IAM":
		if cfg.Provider != nil && cfg.Provider.AWS != nil {
			config.Net.SASL.TokenProvider = &MSKAccessTokenProvider{
				Region:      cfg.Provider.AWS.Region,
				Profile:     cfg.Provider.AWS.Profile,
				RoleARN:     cfg.Provider.AWS.RoleARN,
				SessionName: cfg.Provider.AWS.SessionName,
			}
		}
		config.Net.SASL.Mechanism = sarama.SASLTypeOAuth
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = &tls.Config{}

	default:
		config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
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
		Set("sasl.mechanism", log.String(string(config.Net.SASL.Mechanism))).
		Set("sasl.user", log.String(cfg.Key)).
		Set("topic", log.String(cfg.Topic)).
		Log("setting up kafka subscription")

	return kafkapubsub.OpenSubscription(cfg.Brokers, config, cfg.Group, []string{cfg.Topic}, &kafkapubsub.SubscriptionOptions{
		WaitForJoin: 10 * time.Second,
	})
}

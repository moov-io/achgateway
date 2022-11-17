package awssqs

import (
	"context"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/awssnssqs"
	"time"
)

func OpenSubscription(logger log.Logger, cfg *service.SQSConfig) (*pubsub.Subscription, error) {
	sess, err := session.NewSession(&cfg.Session)
	if err != nil {
		return nil, err
	}

	logger.Info().
		Set("topic_arn", log.String(cfg.TopicARN)).
		Log("setting up sqs subscription")

	return awssnssqs.OpenSubscription(context.TODO(), sess, cfg.TopicARN, &awssnssqs.SubscriptionOptions{
		WaitTime: 10 * time.Second,
	}), nil
}

func OpenTopic(logger log.Logger, cfg *service.SQSConfig) (*pubsub.Topic, error) {
	sess, err := session.NewSession(&cfg.Session)
	if err != nil {
		return nil, err
	}

	logger.Info().
		Set("topic_arn", log.String(cfg.TopicARN)).
		Log("opening sqs topic")

	topic := awssnssqs.OpenSNSTopic(context.TODO(), sess, cfg.TopicARN, nil)

	return topic, nil
}

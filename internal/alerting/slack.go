package alerting

import (
	"errors"
	"fmt"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/slack-go/slack"
)

type Slack struct {
	accessToken string
	channelID   string
	client      *slack.Client
}

func NewSlackAlerter(cfg *service.SlackAlerting) (*Slack, error) {
	notifier := &Slack{
		accessToken: cfg.AccessToken,
		channelID:   cfg.ChannelID,
		client:      slack.New(cfg.AccessToken),
	}
	if err := notifier.AuthTest(); err != nil {
		return nil, err
	}
	return notifier, nil
}

func (s *Slack) Alert(e error) error {
	if e == nil {
		return nil
	}

	_, _, err := s.client.PostMessage(
		s.channelID,
		slack.MsgOptionText(fmt.Sprintf("%v", e), false),
		slack.MsgOptionAsUser(false),
	)
	if err != nil {
		return fmt.Errorf("sending slack message: %v", err)
	}

	return nil
}

func (s *Slack) AlertWithAttachments(msg, color string, fields []slack.AttachmentField) error {
	var attachment = slack.Attachment{
		Fields: fields,
		// color hex value, example: "#8E1600"
		Color: color,
	}

	_, _, err := s.client.PostMessage(
		s.channelID,
		slack.MsgOptionText(msg, false),
		slack.MsgOptionAttachments(attachment),
		slack.MsgOptionAsUser(false),
	)
	if err != nil {
		return fmt.Errorf("sending slack message: %v", err)
	}

	return nil
}

func (s *Slack) AuthTest() error {
	if s == nil || s.client == nil {
		return errors.New("slack: nil or no slack client")
	}

	// make a call and verify we don't error
	resp, err := s.client.AuthTest()
	if err != nil {
		return fmt.Errorf("slack auth test: %v", err)
	}
	if resp.UserID == "" {
		return fmt.Errorf("slack: missing user_id")
	}

	return nil
}

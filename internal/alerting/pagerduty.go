package alerting

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/moov-io/achgateway/internal/service"
)

type PagerDuty struct {
	client     *pagerduty.Client
	routingKey string
}

func NewPagerDutyAlerter(cfg *service.PagerDutyAlerting) (*PagerDuty, error) {
	notifier := &PagerDuty{
		client:     pagerduty.NewClient(cfg.ApiKey),
		routingKey: cfg.RoutingKey,
	}
	if notifier.client != nil {
		notifier.client.SetDebugFlag(pagerduty.DebugCaptureLastResponse)
	}
	if err := notifier.ping(); err != nil {
		return nil, err
	}
	return notifier, nil
}

func (pd *PagerDuty) AlertError(e error) error {
	if e == nil {
		return nil
	}

	hostName, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("getting host name: %v", err)
	}

	details := make(map[string]string)
	for i := 1; i < 5; i++ {
		if _, file, line, ok := runtime.Caller(i); ok {
			caller := fmt.Sprintf("%s:%d", file, line)
			details[fmt.Sprintf("trace_%d", i)] = caller
		}
	}

	dedupKey := e.Error()
	details["dedupKey"] = dedupKey
	errorHash := fmt.Sprintf("%x", sha256.Sum256([]byte(dedupKey)))

	event := &pagerduty.V2Event{
		RoutingKey: pd.routingKey,
		Action:     "trigger",
		DedupKey:   errorHash,
		Payload: &pagerduty.V2Payload{
			Summary:   e.Error(),
			Source:    hostName,
			Severity:  "critical",
			Timestamp: time.Now().Format(time.RFC3339),
			Details:   details,
		},
	}

	if pd.client == nil {
		return errors.New("nil PD client")
	}

	ctx := context.Background()
	v2EventResponse, err := pd.client.ManageEventWithContext(ctx, event)
	if err != nil {
		var httpRespBody []byte
		httpResp, _ := pd.client.LastAPIResponse()
		if httpResp != nil && httpResp.Body != nil {
			httpRespBody, _ = io.ReadAll(httpResp.Body)
		}
		var outErr error
		if v2EventResponse != nil {
			outErr = fmt.Errorf("%s problem creating PagerDuty event caused by %s: %s", v2EventResponse.Status, v2EventResponse.Message, strings.Join(v2EventResponse.Errors, ", "))
		} else {
			outErr = fmt.Errorf("unexpected response of %s from creating event in PagerDuty: %v", string(httpRespBody), err)
		}
		return outErr
	}

	return nil
}

func (pd *PagerDuty) ping() error {
	if pd == nil || pd.client == nil {
		return errors.New("pagerduty: nil")
	}

	// make a call and verify we don't error
	ctx := context.Background()
	resp, err := pd.client.ListAbilitiesWithContext(ctx)
	if err != nil {
		return fmt.Errorf("pagerduty list abilities: %v", err)
	}
	if len(resp.Abilities) <= 0 {
		return fmt.Errorf("pagerduty: missing abilities")
	}

	return nil
}

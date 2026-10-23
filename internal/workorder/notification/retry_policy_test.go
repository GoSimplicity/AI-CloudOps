package notification

import (
	"context"
	"encoding/json"
	"errors"
	"go.uber.org/zap/zaptest/observer"
	"strings"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

func retryInt(value int) *int { return &value }

type retryCountingChannel struct {
	calls    int
	retries  int
	interval time.Duration
	send     func(context.Context, *SendRequest) (*SendResponse, error)
}

func (c *retryCountingChannel) GetName() string                 { return "email" }
func (c *retryCountingChannel) Validate() error                 { return nil }
func (c *retryCountingChannel) IsEnabled() bool                 { return true }
func (c *retryCountingChannel) GetMaxRetries() int              { return c.retries }
func (c *retryCountingChannel) GetRetryInterval() time.Duration { return c.interval }
func (c *retryCountingChannel) Send(ctx context.Context, request *SendRequest) (*SendResponse, error) {
	c.calls++
	if c.send != nil {
		return c.send(ctx, request)
	}
	return nil, errors.New("fake channel unavailable")
}
func retryManager(channel NotificationChannel) *Manager {
	return &Manager{channels: map[string]NotificationChannel{"email": channel}, logger: zap.NewNop()}
}
func retryRequest() *SendRequest {
	return &SendRequest{Channel: "email", MessageID: "stable-message", RecipientAddr: "owned@example.test"}
}
func TestRequestRetryOverrides(t *testing.T) {
	for _, tt := range []struct {
		name              string
		retries, interval *int
		want              int
		invalid           bool
	}{
		{"omitted uses channel", nil, nil, 3, false},
		{"explicit zero", retryInt(0), nil, 1, false},
		{"one extra attempt overrides channel", retryInt(1), retryInt(0), 2, false},
		{"two extra attempts", retryInt(2), retryInt(0), 3, false},
		{"negative retries", retryInt(-1), nil, 0, true},
		{"excessive retries", retryInt(101), nil, 0, true},
		{"negative interval", nil, retryInt(-1), 0, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			channel := &retryCountingChannel{retries: 2}
			request := retryRequest()
			request.MaxRetries = tt.retries
			request.RetryIntervalMinutes = tt.interval
			response, err := retryManager(channel).SendNotification(context.Background(), request)
			if err == nil || channel.calls != tt.want {
				t.Fatalf("calls=%d want=%d err=%v", channel.calls, tt.want, err)
			}
			if !tt.invalid && (response == nil || response.MessageID != request.MessageID) {
				t.Fatal("request MessageID lost")
			}
		})
	}
}
func TestCancelledRequestNeverSends(t *testing.T) {
	channel := &retryCountingChannel{retries: 2}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	response, err := retryManager(channel).SendNotification(ctx, retryRequest())
	if !errors.Is(err, context.Canceled) || channel.calls != 0 || response == nil || response.Status != "cancelled" {
		t.Fatalf("calls=%d response=%v err=%v", channel.calls, response, err)
	}
}
func TestQueuedTaskUsesSingleAttemptAndBothPublishedPayloads(t *testing.T) {
	for _, envelope := range []bool{false, true} {
		t.Run(map[bool]string{false: "legacy raw", true: "request envelope"}[envelope], func(t *testing.T) {
			channel := &retryCountingChannel{retries: 2}
			manager := retryManager(channel)
			request := retryRequest()
			request.MaxRetries = retryInt(2)
			var task *asynq.Task
			if envelope {
				task, _ = CreateSendNotificationTask(request, nil)
			} else {
				task = asynq.NewTask("notification:send", serializeRequest(request))
			}
			err := manager.ProcessNotificationTask(context.Background(), task)
			if err == nil || channel.calls != 1 {
				t.Fatalf("calls=%d err=%v", channel.calls, err)
			}
		})
	}
}

type retryQueue struct {
	calls   int
	task    *asynq.Task
	options []asynq.Option
	ctx     context.Context
	err     error
}

func (q *retryQueue) EnqueueContext(ctx context.Context, task *asynq.Task, options ...asynq.Option) (*asynq.TaskInfo, error) {
	q.calls++
	q.ctx = ctx
	q.task = task
	q.options = options
	return &asynq.TaskInfo{Queue: "default"}, q.err
}
func (*retryQueue) Close() error { return nil }
func TestAsyncRetryOptionsAndWorkerProtocol(t *testing.T) {
	for _, tt := range []struct {
		name     string
		override *int
		want     int
	}{
		{"channel default", nil, 3}, {"explicit zero", retryInt(0), 0}, {"two extra retries", retryInt(2), 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			channel := &retryCountingChannel{retries: 3}
			manager := retryManager(channel)
			queue := &retryQueue{}
			manager.queueClient = queue
			request := retryRequest()
			request.MaxRetries = tt.override
			request.RetryIntervalMinutes = retryInt(7)
			ctx := context.WithValue(context.Background(), struct{}{}, "owner")
			if err := manager.SendNotificationAsync(ctx, request, time.Second); err != nil {
				t.Fatal(err)
			}
			if queue.calls != 1 || queue.ctx != ctx || channel.calls != 0 {
				t.Fatal("enqueue sent notification or discarded context")
			}
			found := map[asynq.OptionType]any{}
			for _, option := range queue.options {
				found[option.Type()] = option.Value()
			}
			if found[asynq.MaxRetryOpt] != tt.want || found[asynq.TaskIDOpt] != request.MessageID || found[asynq.ProcessInOpt] != time.Second {
				t.Fatalf("unexpected queue options %v", found)
			}
			decoded, err := decodeSendNotificationTask(queue.task)
			if err != nil || decoded.MessageID != request.MessageID || decoded.RetryIntervalMinutes == nil || *decoded.RetryIntervalMinutes != 7 {
				t.Fatal("queue retry policy or identity lost")
			}
			if got := manager.RetryDelay(1, errors.New("fake"), queue.task); got != 7*time.Minute {
				t.Fatalf("interval=%s", got)
			}
			if err := NewProcessor(manager, zap.NewNop()).HandleSendNotification(ctx, queue.task); err == nil || channel.calls != 1 {
				t.Fatalf("worker nested retries calls=%d err=%v", channel.calls, err)
			}
		})
	}
}
func TestRetryIntervalConversionAndLimits(t *testing.T) {
	for _, tt := range []struct {
		name    string
		minutes *int
		want    time.Duration
		invalid bool
	}{
		{"channel nanos preserved", nil, 37 * time.Nanosecond, false},
		{"zero interval", retryInt(0), 0, false},
		{"minutes not seconds", retryInt(2), 2 * time.Minute, false},
		{"overflow", retryInt(int(^uint(0) >> 1)), 0, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			request := retryRequest()
			request.RetryIntervalMinutes = tt.minutes
			_, interval, err := requestRetryPolicy(request, &retryCountingChannel{interval: 37 * time.Nanosecond})
			if (err != nil) != tt.invalid || !tt.invalid && interval != tt.want {
				t.Fatalf("interval=%s err=%v", interval, err)
			}
		})
	}
}
func TestRetriesStopAtSuccessAndPreserveMessageID(t *testing.T) {
	for _, failCount := range []int{0, 1, 2} {
		t.Run(string(rune('A'+failCount)), func(t *testing.T) {
			channel := &retryCountingChannel{retries: 3}
			channel.send = func(_ context.Context, request *SendRequest) (*SendResponse, error) {
				if request.MessageID != "stable-message" {
					t.Fatal("attempt message ID changed")
				}
				if channel.calls <= failCount {
					return nil, errors.New("fake failure")
				}
				return &SendResponse{Success: true, MessageID: "provider-id", ExternalID: "external"}, nil
			}
			response, err := retryManager(channel).SendNotification(context.Background(), retryRequest())
			if err != nil || channel.calls != failCount+1 || response.MessageID != "stable-message" || response.ExternalID != "external" {
				t.Fatalf("calls=%d response=%v err=%v", channel.calls, response, err)
			}
		})
	}
}
func TestCancellationInterruptsRetryWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{})
	done := make(chan error, 1)
	channel := &retryCountingChannel{retries: 3, interval: time.Hour, send: func(context.Context, *SendRequest) (*SendResponse, error) {
		close(started)
		return nil, errors.New("fake failure")
	}}
	go func() { _, err := retryManager(channel).SendNotification(ctx, retryRequest()); done <- err }()
	<-started
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) || channel.calls != 1 {
			t.Fatalf("calls=%d err=%v", channel.calls, err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancellation did not interrupt retry wait")
	}
}
func TestAsyncRejectsInvalidRequestBeforeQueueAndPropagatesFailure(t *testing.T) {
	for _, tt := range []struct {
		name      string
		request   *SendRequest
		cancel    bool
		delay     time.Duration
		queueErr  error
		wantQueue int
	}{
		{name: "nil request"},
		{name: "negative retries", request: &SendRequest{Channel: "email", RecipientAddr: "owned@example.test", MaxRetries: retryInt(-1)}},
		{name: "cancelled", request: retryRequest(), cancel: true},
		{name: "negative delay", request: retryRequest(), delay: -time.Second},
		{name: "serialization failure", request: &SendRequest{Channel: "email", RecipientAddr: "owned@example.test", Metadata: map[string]interface{}{"invalid": make(chan int)}}},
		{name: "queue failure", request: retryRequest(), queueErr: context.DeadlineExceeded, wantQueue: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			channel := &retryCountingChannel{}
			manager := retryManager(channel)
			queue := &retryQueue{err: tt.queueErr}
			manager.queueClient = queue
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tt.cancel {
				cancel()
			}
			err := manager.SendNotificationAsync(ctx, tt.request, tt.delay)
			if err == nil || queue.calls != tt.wantQueue || channel.calls != 0 {
				t.Fatalf("queued=%d sent=%d err=%v", queue.calls, channel.calls, err)
			}
			if tt.queueErr != nil && !errors.Is(err, tt.queueErr) {
				t.Fatal("queue error lost")
			}
		})
	}
}
func TestWorkersRejectNullPayloadWithoutSending(t *testing.T) {
	channel := &retryCountingChannel{}
	manager := retryManager(channel)
	processor := NewProcessor(manager, zap.NewNop())
	for _, handler := range []struct {
		name string
		run  func(context.Context, *asynq.Task) error
	}{
		{"manager", manager.ProcessNotificationTask}, {"send", processor.HandleSendNotification}, {"scheduled", processor.HandleScheduledNotification}, {"retry", processor.HandleRetryFailedNotification},
	} {
		t.Run(handler.name, func(t *testing.T) {
			if err := handler.run(context.Background(), asynq.NewTask("notification:send", []byte(`{"request":null}`))); err == nil || channel.calls != 0 {
				t.Fatal("null request accepted or sent")
			}
		})
	}
}
func TestBatchWorkerDoesNotReplaySuccessfulSiblings(t *testing.T) {
	channel := &retryCountingChannel{retries: 3}
	channel.send = func(_ context.Context, request *SendRequest) (*SendResponse, error) {
		if request.MessageID == "fail" {
			return nil, errors.New("fake failed recipient")
		}
		return &SendResponse{Success: true}, nil
	}
	manager := retryManager(channel)
	first, second := retryRequest(), retryRequest()
	first.MessageID = "fail"
	second.MessageID = "success"
	task, err := CreateBatchSendNotificationTask([]*SendRequest{first, second}, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = NewProcessor(manager, zap.NewNop()).HandleBatchSendNotification(context.Background(), task)
	if !errors.Is(err, asynq.SkipRetry) || channel.calls != 2 {
		t.Fatalf("calls=%d err=%v", channel.calls, err)
	}
}

func TestRetryLogsDoNotContainPayloadAddressOrProviderError(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	channel := &retryCountingChannel{send: func(context.Context, *SendRequest) (*SendResponse, error) {
		return nil, errors.New("provider-secret-canary")
	}}
	manager := retryManager(channel)
	manager.logger = zap.New(core)
	request := retryRequest()
	request.Content = "payload-secret-canary"
	request.Metadata = map[string]interface{}{"secret": "metadata-secret-canary"}
	task, err := CreateSendNotificationTask(request, nil)
	if err != nil {
		t.Fatal(err)
	}
	processor := NewProcessor(manager, zap.New(core))
	if err := processor.HandleSendNotification(context.Background(), task); err == nil {
		t.Fatal("fake failure hidden")
	}
	data, err := json.Marshal(logs.All())
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"provider-secret-canary", "payload-secret-canary", "metadata-secret-canary", request.RecipientAddr} {
		if strings.Contains(string(data), secret) {
			t.Fatal("sensitive notification data appeared in log")
		}
	}
}
func TestManagerWithoutQueuePreservesNilBoundary(t *testing.T) {
	manager, err := NewManager(localChannels{}, nil, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	if manager.queueClient != nil {
		t.Fatal("typed nil queue became nonnil interface")
	}
	if err := manager.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestEveryQueueWorkerLeavesRetriesToAsynq(t *testing.T) {
	for _, worker := range []string{"send", "scheduled", "retry"} {
		t.Run(worker, func(t *testing.T) {
			channel := &retryCountingChannel{retries: 4}
			channel.send = func(_ context.Context, request *SendRequest) (*SendResponse, error) {
				if request.MessageID != "stable-message" {
					t.Fatal("worker replaced persisted MessageID")
				}
				return nil, errors.New("fake channel failed")
			}
			processor := NewProcessor(retryManager(channel), zap.NewNop())
			request := retryRequest()
			request.MaxRetries = retryInt(2)
			var task *asynq.Task
			var handle func(context.Context, *asynq.Task) error
			switch worker {
			case "send":
				task, _ = CreateSendNotificationTask(request, nil)
				handle = processor.HandleSendNotification
			case "scheduled":
				task, _ = CreateScheduledNotificationTask(request, time.Now().Add(-time.Minute), "once", nil)
				handle = processor.HandleScheduledNotification
			case "retry":
				task, _ = CreateRetryFailedNotificationTask(request, request.MessageID, 1, "fake", nil)
				handle = processor.HandleRetryFailedNotification
			}
			for i := 1; i <= 2; i++ {
				if err := handle(context.Background(), task); err == nil || channel.calls != i {
					t.Fatalf("calls=%d worker executions=%d err=%v", channel.calls, i, err)
				}
			}
		})
	}
}
func TestBatchManagerHandlesNilAndPreservesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	responses, err := retryManager(&retryCountingChannel{}).BatchSendNotification(ctx, []*SendRequest{nil, retryRequest()})
	if err != nil || len(responses) != 2 || responses[0].Success || responses[1].Status != "cancelled" {
		t.Fatalf("responses=%v err=%v", responses, err)
	}
}

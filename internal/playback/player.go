package playback

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// RequestsPerMinute is the throughput the playback client must finish inside Window.
const (
	RequestsPerMinute = 1000
	Window            = time.Minute
)

// Call is one recorded HTTP request. Playback sends the same value again.
type Call struct {
	Method string
	Path   string
	Body   []byte
}

// Outcome is the last attempt for a call.
type Outcome struct {
	Status   int
	Body     []byte
	Attempts int
	Err      error
}

// Policy controls retries. Only transport errors and HTTP 408, 429, and 5xx are retried.
type Policy struct {
	MaxAttempts    int
	BaseDelay      time.Duration
	MaxDelay       time.Duration
	AttemptTimeout time.Duration
	Concurrency    int
}

func DefaultPolicy() Policy {
	return Policy{MaxAttempts: 6, BaseDelay: 50 * time.Millisecond, MaxDelay: 2 * time.Second, AttemptTimeout: 2 * time.Second, Concurrency: 64}
}

func Retryable(status int) bool {
	return status == http.StatusRequestTimeout || status == http.StatusTooManyRequests || status >= 500
}

func IsIdempotentReplay(status int, body []byte) bool {
	return status == http.StatusOK && bytes.Contains(body, []byte("transaction_id"))
}

// Play sends calls in order. A later Play of the same slice is a playback.
func Play(ctx context.Context, client *http.Client, baseURL string, calls []Call, policy Policy) ([]Outcome, error) {
	if client == nil {
		client = http.DefaultClient
	}
	if policy.MaxAttempts < 1 {
		policy.MaxAttempts = 1
	}
	if policy.MaxAttempts > 6 {
		policy.MaxAttempts = 6
	}
	if policy.Concurrency < 1 {
		policy.Concurrency = 1
	}
	out := make([]Outcome, len(calls))
	jobs := make(chan int)
	var wg sync.WaitGroup
	var firstErr error
	var errMu sync.Mutex
	workers := policy.Concurrency
	if workers > len(calls) {
		workers = len(calls)
	}
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				outcome, err := playOne(ctx, client, baseURL, calls[i], policy)
				out[i] = outcome
				if err != nil {
					errMu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					errMu.Unlock()
				}
			}
		}()
	}
	for i := range calls {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return out, firstErr
}

func playOne(ctx context.Context, client *http.Client, baseURL string, call Call, policy Policy) (Outcome, error) {
	var last Outcome
	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return last, err
		}
		if attempt > 1 && policy.BaseDelay > 0 {
			delay := policy.BaseDelay << (attempt - 2)
			if policy.MaxDelay > 0 && delay > policy.MaxDelay {
				delay = policy.MaxDelay
			}
			if delay > 1 {
				delay = time.Duration(rand.Int63n(int64(delay) + 1))
			}
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return last, ctx.Err()
			case <-timer.C:
			}
		}
		attemptCtx := ctx
		cancelAttempt := func() {}
		if policy.AttemptTimeout > 0 {
			attemptCtx, cancelAttempt = context.WithTimeout(ctx, policy.AttemptTimeout)
		}
		req, err := http.NewRequestWithContext(attemptCtx, call.Method, baseURL+call.Path, bytes.NewReader(call.Body))
		if err != nil {
			cancelAttempt()
			return last, err
		}
		if len(call.Body) > 0 {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := client.Do(req)
		last.Attempts = attempt
		if err != nil {
			cancelAttempt()
			last.Err = err
			last.Status = 0
			if attempt == policy.MaxAttempts || ctx.Err() != nil || (!retryableErr(err) && !errors.Is(err, context.DeadlineExceeded)) {
				return last, nil
			}
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancelAttempt()
		if readErr != nil {
			last.Err = readErr
			last.Status = resp.StatusCode
			if attempt == policy.MaxAttempts || ctx.Err() != nil || (!retryableErr(readErr) && !errors.Is(readErr, context.DeadlineExceeded)) {
				return last, nil
			}
			continue
		}
		last.Status = resp.StatusCode
		last.Body = body
		last.Err = nil
		if !Retryable(resp.StatusCode) || attempt == policy.MaxAttempts || ctx.Err() != nil {
			return last, nil
		}
	}
	return last, nil
}

func retryableErr(err error) bool {
	return err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
}

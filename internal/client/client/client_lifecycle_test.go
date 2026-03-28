package client

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestReportFatalPublishesFirstErrorOnce(t *testing.T) {
	c := &Client{
		exitCh: make(chan error, 1),
	}

	firstErr := errors.New("first failure")
	c.reportFatal(firstErr)
	c.reportFatal(errors.New("second failure"))

	select {
	case err := <-c.Done():
		if !errors.Is(err, firstErr) {
			t.Fatalf("expected first error, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for fatal error")
	}

	select {
	case err := <-c.Done():
		t.Fatalf("unexpected extra error: %v", err)
	default:
	}
}

func TestRunFatalWorkerRecoversPanic(t *testing.T) {
	c := &Client{
		exitCh: make(chan error, 1),
	}

	c.runFatalWorker("test worker", func() error {
		panic("boom")
	})

	select {
	case err := <-c.Done():
		if err == nil {
			t.Fatal("expected panic error, got nil")
		}
		if !strings.Contains(err.Error(), "test worker panic: boom") {
			t.Fatalf("unexpected panic error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for panic error")
	}
}

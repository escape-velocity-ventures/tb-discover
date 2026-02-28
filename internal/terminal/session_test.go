package terminal

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewPTYSession(t *testing.T) {
	var mu sync.Mutex
	var output strings.Builder
	ready := make(chan struct{}, 1)

	onOutput := func(id, data string) {
		mu.Lock()
		defer mu.Unlock()
		output.WriteString(data)
		select {
		case ready <- struct{}{}:
		default:
		}
	}
	onError := func(id, errMsg string) {
		// Allow errors during test cleanup
	}

	session, err := NewPTYSession("test-1", 80, 24, nil, onOutput, onError)
	if err != nil {
		t.Fatalf("NewPTYSession failed: %v", err)
	}
	defer session.Close()

	if session.ID != "test-1" {
		t.Errorf("expected ID test-1, got %s", session.ID)
	}

	// Send a command
	err = session.Write([]byte("echo hello-tb\n"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Wait for output
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for output")
	}

	// Give a moment for all output to arrive
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	got := output.String()
	mu.Unlock()

	if !strings.Contains(got, "hello-tb") {
		t.Errorf("expected output to contain 'hello-tb', got: %s", got)
	}
}

func TestPTYSessionResize(t *testing.T) {
	onOutput := func(id, data string) {}
	onError := func(id, errMsg string) {}

	session, err := NewPTYSession("test-resize", 80, 24, nil, onOutput, onError)
	if err != nil {
		t.Fatalf("NewPTYSession failed: %v", err)
	}
	defer session.Close()

	err = session.Resize(120, 40)
	if err != nil {
		t.Errorf("Resize failed: %v", err)
	}
}

func TestPTYSessionClose(t *testing.T) {
	onOutput := func(id, data string) {}
	onError := func(id, errMsg string) {}

	session, err := NewPTYSession("test-close", 80, 24, nil, onOutput, onError)
	if err != nil {
		t.Fatalf("NewPTYSession failed: %v", err)
	}

	session.Close()

	// Should be safe to close again
	session.Close()

	select {
	case <-session.Done():
		// Good, channel is closed
	case <-time.After(time.Second):
		t.Fatal("Done channel not closed after Close()")
	}
}

func TestPTYSessionEnvFiltering(t *testing.T) {
	// Set sensitive env vars for this test
	t.Setenv("TB_TOKEN", "secret-token-value")
	t.Setenv("AWS_SECRET_KEY", "aws-secret")
	t.Setenv("MY_API_KEY", "some-api-key")
	t.Setenv("DB_PASSWORD", "dbpass")
	t.Setenv("SAFE_VAR", "should-pass")

	var mu sync.Mutex
	var output strings.Builder
	ready := make(chan struct{}, 10)

	onOutput := func(id, data string) {
		mu.Lock()
		defer mu.Unlock()
		output.WriteString(data)
		select {
		case ready <- struct{}{}:
		default:
		}
	}
	onError := func(id, errMsg string) {}

	session, err := NewPTYSession("test-env", 80, 24, nil, onOutput, onError)
	if err != nil {
		t.Fatalf("NewPTYSession failed: %v", err)
	}
	defer session.Close()

	// Print all env vars
	err = session.Write([]byte("env\n"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Wait for output
	time.Sleep(1 * time.Second)

	mu.Lock()
	got := output.String()
	mu.Unlock()

	// Sensitive vars should NOT be present
	for _, sensitive := range []string{"TB_TOKEN=", "AWS_SECRET_KEY=", "MY_API_KEY=", "DB_PASSWORD="} {
		if strings.Contains(got, sensitive) {
			t.Errorf("PTY env should not contain %s", sensitive)
		}
	}

	// TERM should be set
	if !strings.Contains(got, "TERM=xterm-256color") {
		t.Errorf("PTY env should contain TERM=xterm-256color")
	}
}

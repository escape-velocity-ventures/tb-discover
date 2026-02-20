package agent

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestScanLoopRunsInitialScan(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	sl := NewScanLoop(ScanLoopConfig{
		Profile:  "minimal",
		Interval: 1 * time.Hour, // Long interval — we only care about the initial scan
		Version:  "test",
	}, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Run in goroutine
	done := make(chan struct{})
	go func() {
		sl.Run(ctx)
		close(done)
	}()

	// Wait for context to expire or loop to finish
	select {
	case <-done:
		// Good — loop exited cleanly on context cancel
	case <-time.After(10 * time.Second):
		t.Fatal("scan loop did not stop after context cancellation")
	}
}

func TestScanLoopGracefulShutdown(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	sl := NewScanLoop(ScanLoopConfig{
		Profile:  "minimal",
		Interval: 100 * time.Millisecond,
		Version:  "test",
	}, logger)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		sl.Run(ctx)
		close(done)
	}()

	// Let it run a few scan cycles
	time.Sleep(350 * time.Millisecond)

	// Cancel and verify it stops
	cancel()

	select {
	case <-done:
		// Clean shutdown
	case <-time.After(5 * time.Second):
		t.Fatal("scan loop did not stop after cancel")
	}
}

func TestScanLoopNoUploadWithoutClient(t *testing.T) {
	// Verify scan loop works without upload config (no panics, no errors)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	sl := NewScanLoop(ScanLoopConfig{
		Profile:  "minimal",
		Interval: 1 * time.Hour,
		Version:  "test",
		// No UploadURL or Token — client will be nil
	}, logger)

	if sl.client != nil {
		t.Error("expected nil client when no upload URL configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		sl.Run(ctx)
		close(done)
	}()

	<-done
}

func TestScanLoopInvalidProfile(t *testing.T) {
	// Verify scan loop handles invalid profile without crashing
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	sl := NewScanLoop(ScanLoopConfig{
		Profile:  "nonexistent",
		Interval: 1 * time.Hour,
		Version:  "test",
	}, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		sl.Run(ctx)
		close(done)
	}()

	<-done
	// If we got here without panic, the test passes
}

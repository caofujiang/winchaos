package tools

import (
	"context"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

func init() {
	app.ctx, app.cancel = context.WithCancel(context.Background())
	app.wgDone = make(chan struct{})
	go closeAppWhenAnExitSignalIsEncountered()
}

var app struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	wgDone chan struct{}

	mu     sync.Mutex
	defers []func()
}

func Go(ctx context.Context, fn func(ctx context.Context, closing <-chan struct{})) {
	app.wg.Add(1)

	go func(ctx context.Context, closing <-chan struct{}, wg *sync.WaitGroup, fn func(ctx context.Context, closing <-chan struct{})) {
		defer wg.Done()

		fn(ctx, closing)
	}(ctx, Closing(), &app.wg, fn)
}

func Defer(fn func()) {
	if fn == nil {
		return
	}
	app.mu.Lock()
	app.defers = append(app.defers, fn)
	app.mu.Unlock()
}

func Wait() {
	defer close(app.wgDone)

	app.wg.Wait()

	// call defers
	app.mu.Lock()
	defer app.mu.Unlock()
	for i := len(app.defers) - 1; i >= 0; i-- {
		app.defers[i]()
	}
}

func Close() { app.cancel() }

func Closing() <-chan struct{} { return app.ctx.Done() }

func closeAppWhenAnExitSignalIsEncountered() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	sig := <-sigCh
	logrus.Info("got-exit-signal-and-closing", "signal", sig)
	Close()

	timer := time.NewTimer(30 * time.Second)
	select {
	case <-timer.C:
		//  panic for error
		if _, err := os.Stderr.WriteString("panic: graceful_shutdown_timeout"); err != nil {
			logrus.Fatal("graceful_shutdown_timeout")
		}
		debug.PrintStack()
		os.Exit(1)
	case <-app.wgDone:
		timer.Stop()
	}
}

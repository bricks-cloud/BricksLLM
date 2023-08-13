package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/logger/zap"
	"github.com/bricks-cloud/bricksllm/internal/manager"
	"github.com/bricks-cloud/bricksllm/internal/server/web"
	"github.com/bricks-cloud/bricksllm/internal/storage/postgresql"
)

func main() {
	modePtr := flag.String("m", "dev", "select the mode that bricksllm runs in")
	lg := zap.NewLogger(*modePtr)

	store, err := postgresql.NewStore("", lg)
	if err != nil {
		lg.Fatalf("cannot connect to postgresql: %v", err)
	}

	err = store.CreateKeysTable()
	if err != nil {
		lg.Fatalf("error creating keys table: %v", err)
	}

	m := manager.NewManager(store)
	as, err := web.NewAdminServer(lg, m)
	if err != nil {
		lg.Fatalf("error creating http server: %v", err)
	}

	as.Run()

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	lg.Infof("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := as.Shutdown(ctx); err != nil {
		lg.Fatalf("server shutdown: %v", err)
	}

	select {
	case <-ctx.Done():
		lg.Infof("timeout of 5 seconds")
	}

	err = store.DropKeysTable()
	if err != nil {
		lg.Fatalf("error dropping keys table: %v", err)
	}

	lg.Infof("server exited")
}

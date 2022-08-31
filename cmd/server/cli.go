package server

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	api "eth-analyse-service/pkg"
)

func RunServer() int {
	e := api.New()
	go func() {
		if err := e.Start(":8805"); err != nil {
			e.Logger.Fatalf("Shutting down..", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT)
	<-quit // Blocking until signal arrives

	ctx, ctxCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer ctxCancel()
	if err := e.Shutdown(ctx); err != nil {
		return 1
	}

	return 0
}

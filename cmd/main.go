package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func main() {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		panic(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	for {
		runCtx, runCancel := context.WithCancel(ctx)
		go handleEvents(ctx, runCancel, cli)
		<-runCtx.Done()

		select {
		case <-ctx.Done():
			return
		default:
		}

		fmt.Printf("will try to reconnect after 1 sec\n")
		time.Sleep(time.Second)
	}
}

func handleEvents(ctx context.Context, cancel func(), cli *client.Client) {
	defer cancel()

	fmt.Printf("start listen events\n")
	msgCh, errCh := cli.Events(ctx, types.EventsOptions{})
	for {
		select {
		case ev := <-msgCh:
			fmt.Printf("captured event: %+v\n", ev)
		case ev := <-errCh:
			fmt.Printf("captured error event %v\n", ev)
			return
		case <-ctx.Done():
			fmt.Printf("captured stop event, shut down\n")
			return
		}
	}
}

package main

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"io"
	"log"
	"os"
)

func spawn(local bool, id string) {
	log.Println("Spawning a new recording container")
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	if local == false {
		reader, err := cli.ImagePull(ctx, "recorder", types.ImagePullOptions{})
		if err != nil {
			fmt.Fprint(os.Stderr, err)
		}
		io.Copy(os.Stdout, reader)
	}
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Env:   []string{fmt.Sprintf("key=%s", id)},
		Image: "recorder",
	}, &container.HostConfig{
		NetworkMode: "host",
		Binds: []string{
			"/home/denzyl/test:/home/code",
		},
	}, nil, nil, "")
	if err != nil {
		fmt.Fprint(os.Stderr, err)

	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	/// @todo use the dataPasser struct.
	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case <-statusCh:
	}
}

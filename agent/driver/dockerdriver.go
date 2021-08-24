// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package driver

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"io"
)

type DockerDriver struct {
	client            *client.Client
	containerNameOrId string
	user              string
}

func NewDockerDriver(containerIdOrName, user string) (*DockerDriver, error) {
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &DockerDriver{client: client, containerNameOrId: containerIdOrName, user: user}, nil
}

type Error struct {
	errorCode int
}

func (e *Error) Error() string {
	return fmt.Sprintf("Command exited with error code %v", e.errorCode)
}

func (d *DockerDriver) ExecuteCommand(ctx context.Context, command []string, stdin io.Reader, stdout, stderr io.Writer) error {
	config := types.ExecConfig{
		User:         d.user,
		AttachStdout: true,
		AttachStderr: true,
		AttachStdin:  stdin != nil,
		Detach:       false,
		Cmd:          command,
		WorkingDir:   "/",
		Tty:          false,
	}

	idResponse, err := d.client.ContainerExecCreate(ctx, d.containerNameOrId, config)
	if err != nil {
		return errorutil.Wrap(err)
	}

	execStartCheck := types.ExecStartCheck{
		Detach: false,
		Tty:    false,
	}

	response, err := d.client.ContainerExecAttach(ctx, idResponse.ID, execStartCheck)
	if err != nil {
		return errorutil.Wrap(err)
	}

	defer response.Close()

	doneReading := make(chan error, 1)
	doneWriting := make(chan error, 1)

	go func() {
		if stdin == nil {
			doneWriting <- nil
			return
		}

		_, err := io.Copy(response.Conn, stdin)

		if err := response.CloseWrite(); err != nil {
			doneWriting <- err
			return
		}

		doneWriting <- err
	}()

	go func() {
		_, err := stdcopy.StdCopy(stdout, stderr, response.Reader)
		doneReading <- err
	}()

	if err := d.client.ContainerExecStart(ctx, idResponse.ID, execStartCheck); err != nil {
		return errorutil.Wrap(err)
	}

	inspect, err := d.client.ContainerExecInspect(ctx, idResponse.ID)
	if err != nil {
		return errorutil.Wrap(err)
	}

	if inspect.ExitCode != 0 {
		return &Error{errorCode: inspect.ExitCode}
	}

	for _, c := range []chan error{doneWriting, doneReading} {
		select {
		case err := <-c:
			if err != nil {
				return errorutil.Wrap(err)
			}
		case <-ctx.Done():
			return errorutil.Wrap(ctx.Err())
		}
	}

	return nil
}

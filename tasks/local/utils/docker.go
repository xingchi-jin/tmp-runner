package utils

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

var (
	docker *Docker
)

type Docker struct {
	client *client.Client
}

// GetDockerClient returns a singleton object of Docker
func GetDockerClient() (*Docker, error) {
	if docker == nil {
		if cli, err := client.NewClientWithOpts(client.FromEnv); err != nil {
			return nil, err
		} else {
			docker = &Docker{client: cli}
		}
	}
	return docker, nil
}

// DestroyContainersByLabel destroys a pipeline config and cleans up all containers with
// a if specified. This should be used in favor of the old Destroy() which is stateful.
func (d *Docker) KillContainersByLabel(
	ctx context.Context,
	labels map[string]string,
) error {
	args := filters.NewArgs()
	if len(labels) > 0 {
		for labelKey, labelValue := range labels {
			args.Add("label", fmt.Sprintf("%s=%s", labelKey, labelValue))
		}
	} else {
		// If no labels provided, don't kill anything
		return nil
	}
	ctrs, err := d.client.ContainerList(ctx, types.ContainerListOptions{
		Filters: args,
		All:     true,
	})
	logrus.Info(ctrs)
	if err != nil {
		return err
	}
	var containers []string
	for i := range ctrs {
		containers = append(containers, ctrs[i].ID)
	}
	return d.KillContainers(ctx, containers)
}

func (d *Docker) KillContainers(ctx context.Context, containerIds []string) error {
	// stop all containers. Soft stop feature should be in a different function
	for _, id := range containerIds {
		if err := d.client.ContainerKill(ctx, id, "9"); err != nil {
			logrus.WithField("container", id).WithField("error", err).Warnln("failed to kill container")
		}
	}

	removeOpts := types.ContainerRemoveOptions{
		Force:         true,
		RemoveLinks:   false,
		RemoveVolumes: true,
	}
	// cleanup all containers
	for _, id := range containerIds {
		if err := d.client.ContainerRemove(ctx, id, removeOpts); err != nil {
			logrus.WithField("container", id).WithField("error", err).Warnln("failed to remove container")
			return err
		}
	}
	return nil
}

func (d *Docker) RemoveNetworks(ctx context.Context, ids []string) error {
	for _, id := range ids {
		// cleanup the network
		if err := d.client.NetworkRemove(ctx, id); err != nil {
			logrus.WithField("network", id).WithField("error", err).Warnln("failed to remove network")
			return err
		}
	}
	return nil
}

// A function to sanitize any string and make it compatible with docker
func Sanitize(id string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return r
		}
		return '_'
	}, id)
}

func GeneratePath(id string) string {
	return fmt.Sprintf("/tmp/harness/%s", Sanitize(id))
}

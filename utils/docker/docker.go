package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"

	"github.com/aihop/gopanel/global"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"

	"github.com/docker/docker/client"
)

type Client struct {
	cli *client.Client
}

type networkRuntimeClient interface {
	Close()
	NetworkExist(name string) bool
	CreateNetwork(name string) error
}

func NewRuntimeClient() (Client, error) {
	cli, err := DefaultRuntimeAdapter().DockerClient(context.Background())
	if err != nil {
		return Client{}, err
	}

	return Client{
		cli: cli,
	}, nil
}

func NewClient() (Client, error) {
	return NewRuntimeClient()
}

func (c Client) Close() {
	_ = c.cli.Close()
}

func NewRuntimeAPIClient() (*client.Client, error) {
	return DefaultRuntimeAdapter().DockerClient(context.Background())
}

func NewDockerClient() (*client.Client, error) {
	return NewRuntimeAPIClient()
}

func (c Client) ListContainersByName(names []string) ([]types.Container, error) {
	var (
		options  container.ListOptions
		namesMap = make(map[string]bool)
		res      []types.Container
	)
	options.All = true
	if len(names) > 0 {
		var array []filters.KeyValuePair
		for _, n := range names {
			namesMap["/"+n] = true
			array = append(array, filters.Arg("name", n))
		}
		options.Filters = filters.NewArgs(array...)
	}
	containers, err := c.cli.ContainerList(context.Background(), options)
	if err != nil {
		return nil, err
	}
	for _, con := range containers {
		if _, ok := namesMap[con.Names[0]]; ok {
			res = append(res, con)
		}
	}
	return res, nil
}
func (c Client) ListAllContainers() ([]types.Container, error) {
	var (
		options container.ListOptions
	)
	options.All = true
	containers, err := c.cli.ContainerList(context.Background(), options)
	if err != nil {
		return nil, err
	}
	return containers, nil
}

func (c Client) CreateNetwork(name string) error {
	_, err := c.cli.NetworkCreate(context.Background(), name, network.CreateOptions{
		Driver: "bridge",
	})
	return err
}

func (c Client) DeleteImage(imageID string) error {
	if _, err := c.cli.ImageRemove(context.Background(), imageID, image.RemoveOptions{Force: true}); err != nil {
		return err
	}
	return nil
}

func (c Client) InspectContainer(containerID string) (types.ContainerJSON, error) {
	return c.cli.ContainerInspect(context.Background(), containerID)
}

func (c Client) PullImage(imageName string, force bool) error {
	if !force {
		exist, err := c.CheckImageExist(imageName)
		if err != nil {
			return err
		}
		if exist {
			return nil
		}
	}
	if _, err := c.cli.ImagePull(context.Background(), imageName, image.PullOptions{}); err != nil {
		return err
	}
	return nil
}

func (c Client) GetImageIDByName(imageName string) (string, error) {
	filter := filters.NewArgs()
	filter.Add("reference", imageName)
	list, err := c.cli.ImageList(context.Background(), image.ListOptions{
		Filters: filter,
	})
	if err != nil {
		return "", err
	}
	if len(list) > 0 {
		return list[0].ID, nil
	}
	return "", nil
}

func (c Client) CheckImageExist(imageName string) (bool, error) {
	filter := filters.NewArgs()
	filter.Add("reference", imageName)
	list, err := c.cli.ImageList(context.Background(), image.ListOptions{
		Filters: filter,
	})
	if err != nil {
		return false, err
	}
	return len(list) > 0, nil
}

func (c Client) NetworkExist(name string) bool {
	var options network.ListOptions
	options.Filters = filters.NewArgs(filters.Arg("name", name))
	networks, err := c.cli.NetworkList(context.Background(), options)
	if err != nil {
		return false
	}
	return len(networks) > 0
}

func CreateDefaultDockerNetwork() error {
	resolved := ResolveRuntime(context.Background())
	if resolved.Kind == RuntimePodman {
		if err := ensurePodmanNetworkWithFallback("gopanel-network", ensurePodmanNetwork, newNetworkRuntimeClient); err != nil {
			global.LOG.Errorf("create default podman network error %s", err.Error())
			return err
		}
		return nil
	}
	cli, err := NewClient()
	if err != nil {
		global.LOG.Errorf("init docker client error %s", err.Error())
		return err
	}
	defer cli.Close()
	if !cli.NetworkExist("gopanel-network") {
		if err := cli.CreateNetwork("gopanel-network"); err != nil {
			global.LOG.Errorf("create default docker network  error %s", err.Error())
			return err
		}
	}
	return nil
}

func EnsureNetwork(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	resolved := ResolveRuntime(context.Background())
	if resolved.Kind == RuntimePodman {
		return ensurePodmanNetworkWithFallback(name, ensurePodmanNetwork, newNetworkRuntimeClient)
	}
	cli, err := NewClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	if !cli.NetworkExist(name) {
		return cli.CreateNetwork(name)
	}
	return nil
}

func newNetworkRuntimeClient() (networkRuntimeClient, error) {
	return NewClient()
}

func ensurePodmanNetworkWithFallback(name string, ensureCLI func(string) error, newClient func() (networkRuntimeClient, error)) error {
	cliErr := ensureCLI(name)
	if cliErr == nil {
		return nil
	}
	apiClient, err := newClient()
	if err != nil {
		return fmt.Errorf("Podman CLI 网络准备失败: %v；Socket API 初始化失败: %w", cliErr, err)
	}
	defer apiClient.Close()
	if apiClient.NetworkExist(name) {
		return nil
	}
	if err := apiClient.CreateNetwork(name); err != nil {
		return fmt.Errorf("Podman CLI 网络准备失败: %v；Socket API 创建网络失败: %w", cliErr, err)
	}
	return nil
}

func ensurePodmanNetwork(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	ctx := context.Background()
	if err := PodmanEnsureReady(ctx); err != nil {
		return err
	}
	if _, err := runPodman(ctx, "network", "inspect", name); err == nil {
		return nil
	}
	out, err := runPodman(ctx, "network", "create", name)
	if err != nil {
		if strings.TrimSpace(out) != "" {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(out))
		}
		return err
	}
	return nil
}

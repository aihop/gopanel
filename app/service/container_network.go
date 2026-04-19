package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/docker/docker/api/types/network"
)

func (u *ContainerService) PageNetwork(req *dto.SearchWithPage) (int64, interface{}, error) {
	if docker.IsPodmanRuntime(context.Background()) && runtime.GOOS == "darwin" {
		return u.pageNetworkPodman(req)
	}
	client, err := docker.NewDockerClient()
	if err != nil {
		return 0, nil, err
	}
	defer client.Close()
	list, err := client.NetworkList(context.TODO(), network.ListOptions{})
	if err != nil {
		return 0, nil, err
	}
	if len(req.Info) != 0 {
		length, count := len(list), 0
		for count < length {
			if !strings.Contains(list[count].Name, req.Info) {
				list = append(list[:count], list[(count+1):]...)
				length--
			} else {
				count++
			}
		}
	}
	var (
		data    []dto.Network
		records []network.Inspect
	)
	sort.Slice(list, func(i, j int) bool {
		return list[i].Created.Before(list[j].Created)
	})
	total, start, end := len(list), (req.Page-1)*req.Limit, req.Page*req.Limit
	if start > total {
		records = make([]network.Inspect, 0)
	} else {
		if end >= total {
			end = total
		}
		records = list[start:end]
	}

	for _, item := range records {
		tag := make([]string, 0)
		for key, val := range item.Labels {
			tag = append(tag, fmt.Sprintf("%s=%s", key, val))
		}
		var ipam network.IPAMConfig
		if len(item.IPAM.Config) > 0 {
			ipam = item.IPAM.Config[0]
		}
		data = append(data, dto.Network{
			ID:         item.ID,
			CreatedAt:  item.Created,
			Name:       item.Name,
			Driver:     item.Driver,
			IPAMDriver: item.IPAM.Driver,
			Subnet:     ipam.Subnet,
			Gateway:    ipam.Gateway,
			Attachable: item.Attachable,
			Labels:     tag,
		})
	}

	return int64(total), data, nil
}

func (u *ContainerService) ListNetwork() ([]dto.Options, error) {
	if docker.IsPodmanRuntime(context.Background()) && runtime.GOOS == "darwin" {
		return u.listNetworkPodman()
	}
	client, err := docker.NewDockerClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()
	list, err := client.NetworkList(context.TODO(), network.ListOptions{})
	if err != nil {
		return nil, err
	}
	var datas []dto.Options
	for _, item := range list {
		datas = append(datas, dto.Options{Option: item.Name})
	}
	sort.Slice(datas, func(i, j int) bool {
		return datas[i].Option < datas[j].Option
	})
	return datas, nil
}

func (u *ContainerService) DeleteNetwork(req *dto.BatchDelete) error {
	if docker.IsPodmanRuntime(context.Background()) && runtime.GOOS == "darwin" {
		return u.deleteNetworkPodman(req)
	}
	client, err := docker.NewDockerClient()
	if err != nil {
		return err
	}
	defer client.Close()
	for _, id := range req.Names {
		if err := client.NetworkRemove(context.TODO(), id); err != nil {
			if strings.Contains(err.Error(), "has active endpoints") {
				return buserr.WithDetail(constant.ErrInUsed, id, nil)
			}
			return err
		}
	}
	return nil
}
func (u *ContainerService) CreateNetwork(req *dto.NetworkCreate) error {
	if docker.IsPodmanRuntime(context.Background()) && runtime.GOOS == "darwin" {
		return u.createNetworkPodman(req)
	}
	client, err := docker.NewDockerClient()
	if err != nil {
		return err
	}
	defer client.Close()
	var (
		ipams    []network.IPAMConfig
		enableV6 bool
	)
	if req.Ipv4 {
		var itemIpam network.IPAMConfig
		if len(req.AuxAddress) != 0 {
			itemIpam.AuxAddress = make(map[string]string)
		}
		if len(req.Subnet) != 0 {
			itemIpam.Subnet = req.Subnet
		}
		if len(req.Gateway) != 0 {
			itemIpam.Gateway = req.Gateway
		}
		if len(req.IPRange) != 0 {
			itemIpam.IPRange = req.IPRange
		}
		for _, addr := range req.AuxAddress {
			itemIpam.AuxAddress[addr.Key] = addr.Value
		}
		ipams = append(ipams, itemIpam)
	}
	if req.Ipv6 {
		enableV6 = true
		var itemIpam network.IPAMConfig
		if len(req.AuxAddress) != 0 {
			itemIpam.AuxAddress = make(map[string]string)
		}
		if len(req.SubnetV6) != 0 {
			itemIpam.Subnet = req.SubnetV6
		}
		if len(req.GatewayV6) != 0 {
			itemIpam.Gateway = req.GatewayV6
		}
		if len(req.IPRangeV6) != 0 {
			itemIpam.IPRange = req.IPRangeV6
		}
		for _, addr := range req.AuxAddressV6 {
			itemIpam.AuxAddress[addr.Key] = addr.Value
		}
		ipams = append(ipams, itemIpam)
	}

	options := network.CreateOptions{
		EnableIPv6: &enableV6,
		Driver:     req.Driver,
		Options:    stringsToMap(req.Options),
		Labels:     stringsToMap(req.Labels),
	}
	if len(ipams) != 0 {
		options.IPAM = &network.IPAM{Config: ipams}
	}
	if _, err := client.NetworkCreate(context.TODO(), req.Name, options); err != nil {
		return err
	}
	return nil
}

func (u *ContainerService) pageNetworkPodman(req *dto.SearchWithPage) (int64, interface{}, error) {
	if err := docker.PodmanEnsureReady(context.Background()); err != nil {
		return 0, nil, err
	}
	cmdExec := exec.Command("podman", "network", "ls", "--format", "json")
	out, err := cmdExec.CombinedOutput()
	if err != nil {
		return 0, nil, errors.New(strings.TrimSpace(string(out)))
	}
	var raw []map[string]interface{}
	if err := json.Unmarshal(out, &raw); err != nil {
		return 0, nil, err
	}
	var list []dto.Network
	for _, item := range raw {
		name := strings.TrimSpace(fmt.Sprint(item["name"]))
		if name == "" || name == "<nil>" {
			name = strings.TrimSpace(fmt.Sprint(item["Name"]))
		}
		if req.Info != "" && !strings.Contains(name, req.Info) {
			continue
		}
		n := dto.Network{
			ID:     strings.TrimSpace(fmt.Sprint(item["id"])),
			Name:   name,
			Driver: strings.TrimSpace(fmt.Sprint(item["driver"])),
		}
		if n.ID == "" || n.ID == "<nil>" {
			n.ID = strings.TrimSpace(fmt.Sprint(item["ID"]))
		}
		if n.Driver == "" || n.Driver == "<nil>" {
			n.Driver = strings.TrimSpace(fmt.Sprint(item["Driver"]))
		}
		if labels, ok := item["labels"].(map[string]interface{}); ok {
			for k, v := range labels {
				n.Labels = append(n.Labels, fmt.Sprintf("%s=%v", k, v))
			}
		}
		if labels, ok := item["Labels"].(map[string]interface{}); ok && len(n.Labels) == 0 {
			for k, v := range labels {
				n.Labels = append(n.Labels, fmt.Sprintf("%s=%v", k, v))
			}
		}

		if ipamOpts, ok := item["ipam_options"].(map[string]interface{}); ok {
			n.IPAMDriver = strings.TrimSpace(fmt.Sprint(ipamOpts["driver"]))
		}
		if subnets, ok := item["subnets"].([]interface{}); ok && len(subnets) > 0 {
			if s0, ok := subnets[0].(map[string]interface{}); ok {
				n.Subnet = strings.TrimSpace(fmt.Sprint(s0["subnet"]))
				n.Gateway = strings.TrimSpace(fmt.Sprint(s0["gateway"]))
			}
		}
		if created := strings.TrimSpace(fmt.Sprint(item["created"])); created != "" && created != "<nil>" {
			if t, e := time.Parse(time.RFC3339Nano, created); e == nil {
				n.CreatedAt = t
			}
		}
		list = append(list, n)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt.Before(list[j].CreatedAt) })
	total := len(list)
	start, end := (req.Page-1)*req.Limit, req.Page*req.Limit
	if start < 0 {
		start = 0
	}
	if end > total {
		end = total
	}
	if start > total {
		return int64(total), []dto.Network{}, nil
	}
	return int64(total), list[start:end], nil
}

func (u *ContainerService) listNetworkPodman() ([]dto.Options, error) {
	if err := docker.PodmanEnsureReady(context.Background()); err != nil {
		return nil, err
	}
	cmdExec := exec.Command("podman", "network", "ls", "--format", "json")
	out, err := cmdExec.CombinedOutput()
	if err != nil {
		return nil, errors.New(strings.TrimSpace(string(out)))
	}
	var raw []map[string]interface{}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, err
	}
	var datas []dto.Options
	for _, item := range raw {
		name := strings.TrimSpace(fmt.Sprint(item["name"]))
		if name == "" || name == "<nil>" {
			name = strings.TrimSpace(fmt.Sprint(item["Name"]))
		}
		if name == "" || name == "<nil>" {
			continue
		}
		datas = append(datas, dto.Options{Option: name})
	}
	sort.Slice(datas, func(i, j int) bool { return datas[i].Option < datas[j].Option })
	return datas, nil
}

func (u *ContainerService) deleteNetworkPodman(req *dto.BatchDelete) error {
	if err := docker.PodmanEnsureReady(context.Background()); err != nil {
		return err
	}
	for _, id := range req.Names {
		c := exec.Command("podman", "network", "rm", id)
		out, err := c.CombinedOutput()
		if err != nil {
			msg := strings.TrimSpace(string(out))
			if strings.Contains(strings.ToLower(msg), "in use") || strings.Contains(strings.ToLower(msg), "active endpoints") {
				return buserr.WithDetail(constant.ErrInUsed, id, nil)
			}
			if msg == "" {
				return err
			}
			return errors.New(msg)
		}
	}
	return nil
}

func (u *ContainerService) createNetworkPodman(req *dto.NetworkCreate) error {
	if err := docker.PodmanEnsureReady(context.Background()); err != nil {
		return err
	}
	args := []string{"network", "create", "--driver", req.Driver}
	if req.Ipv6 {
		args = append(args, "--ipv6")
	}
	if req.Subnet != "" {
		args = append(args, "--subnet", req.Subnet)
	}
	if req.Gateway != "" {
		args = append(args, "--gateway", req.Gateway)
	}
	if req.IPRange != "" {
		args = append(args, "--ip-range", req.IPRange)
	}
	if req.SubnetV6 != "" {
		args = append(args, "--subnet", req.SubnetV6)
	}
	if req.GatewayV6 != "" {
		args = append(args, "--gateway", req.GatewayV6)
	}
	if req.IPRangeV6 != "" {
		args = append(args, "--ip-range", req.IPRangeV6)
	}
	for k, v := range stringsToMap(req.Options) {
		args = append(args, "--opt", fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range stringsToMap(req.Labels) {
		args = append(args, "--label", fmt.Sprintf("%s=%s", k, v))
	}
	args = append(args, req.Name)
	c := exec.Command("podman", args...)
	out, err := c.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return err
		}
		return errors.New(msg)
	}
	return nil
}

package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/common"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
)

func (u *ContainerService) PageVolume(req *dto.SearchWithPage) (int64, interface{}, error) {
	if docker.IsPodmanRuntime(context.Background()) {
		return u.pageVolumePodman(req)
	}
	client, err := docker.NewDockerClient()
	if err != nil {
		return 0, nil, err
	}
	list, err := client.VolumeList(context.TODO(), volume.ListOptions{})
	if err != nil {
		return 0, nil, err
	}
	if len(req.Info) != 0 {
		length, count := len(list.Volumes), 0
		for count < length {
			if !strings.Contains(list.Volumes[count].Name, req.Info) {
				list.Volumes = append(list.Volumes[:count], list.Volumes[(count+1):]...)
				length--
			} else {
				count++
			}
		}
	}
	var (
		data    []dto.Volume
		records []*volume.Volume
	)
	sort.Slice(list.Volumes, func(i, j int) bool {
		return list.Volumes[i].CreatedAt > list.Volumes[j].CreatedAt
	})
	total, start, end := len(list.Volumes), (req.Page-1)*req.Limit, req.Page*req.Limit
	if start > total {
		records = make([]*volume.Volume, 0)
	} else {
		if end >= total {
			end = total
		}
		records = list.Volumes[start:end]
	}

	nyc, _ := time.LoadLocation(common.LoadTimeZoneByCmd())
	for _, item := range records {
		tag := make([]string, 0)
		for _, val := range item.Labels {
			tag = append(tag, val)
		}
		var createTime time.Time
		if strings.Contains(item.CreatedAt, "Z") {
			createTime, _ = time.ParseInLocation("2006-01-02T15:04:05Z", item.CreatedAt, nyc)
		} else if strings.Contains(item.CreatedAt, "+") {
			createTime, _ = time.ParseInLocation("2006-01-02T15:04:05+08:00", item.CreatedAt, nyc)
		} else {
			createTime, _ = time.ParseInLocation("2006-01-02T15:04:05", item.CreatedAt, nyc)
		}
		data = append(data, dto.Volume{
			CreatedAt:  createTime,
			Name:       item.Name,
			Driver:     item.Driver,
			Mountpoint: item.Mountpoint,
			Labels:     tag,
		})
	}

	return int64(total), data, nil
}
func (u *ContainerService) ListVolume() ([]dto.Options, error) {
	if docker.IsPodmanRuntime(context.Background()) {
		return u.listVolumePodman()
	}
	client, err := docker.NewDockerClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()
	list, err := client.VolumeList(context.TODO(), volume.ListOptions{})
	if err != nil {
		return nil, err
	}
	var datas []dto.Options
	for _, item := range list.Volumes {
		datas = append(datas, dto.Options{
			Option: item.Name,
		})
	}
	sort.Slice(datas, func(i, j int) bool {
		return datas[i].Option < datas[j].Option
	})
	return datas, nil
}
func (u *ContainerService) DeleteVolume(req *dto.BatchDelete) error {
	if docker.IsPodmanRuntime(context.Background()) {
		return u.deleteVolumePodman(req)
	}
	client, err := docker.NewDockerClient()
	if err != nil {
		return err
	}
	defer client.Close()
	for _, id := range req.Names {
		if err := client.VolumeRemove(context.TODO(), id, true); err != nil {
			if strings.Contains(err.Error(), "volume is in use") {
				return buserr.WithDetail(constant.ErrInUsed, id, nil)
			}
			return err
		}
	}
	return nil
}
func (u *ContainerService) CreateVolume(req *dto.VolumeCreate) error {
	if docker.IsPodmanRuntime(context.Background()) {
		return u.createVolumePodman(req)
	}
	client, err := docker.NewDockerClient()
	if err != nil {
		return err
	}
	defer client.Close()
	arg := filters.NewArgs()
	arg.Add("name", req.Name)
	vos, _ := client.VolumeList(context.TODO(), volume.ListOptions{Filters: arg})
	if len(vos.Volumes) != 0 {
		for _, v := range vos.Volumes {
			if v.Name == req.Name {
				return constant.ErrRecordExist
			}
		}
	}
	options := volume.CreateOptions{
		Name:       req.Name,
		Driver:     req.Driver,
		DriverOpts: stringsToMap(req.Options),
		Labels:     stringsToMap(req.Labels),
	}
	if _, err := client.VolumeCreate(context.TODO(), options); err != nil {
		return err
	}
	return nil
}

func (u *ContainerService) pageVolumePodman(req *dto.SearchWithPage) (int64, interface{}, error) {
	if err := docker.PodmanEnsureReady(context.Background()); err != nil {
		return 0, nil, err
	}
	cmdExec := exec.Command("podman", "volume", "ls", "--format", "json")
	out, err := cmdExec.CombinedOutput()
	if err != nil {
		return 0, nil, errors.New(strings.TrimSpace(string(out)))
	}
	var raw []map[string]interface{}
	if err := json.Unmarshal(out, &raw); err != nil {
		return 0, nil, err
	}
	var list []dto.Volume
	for _, item := range raw {
		name := strings.TrimSpace(fmt.Sprint(item["Name"]))
		if req.Info != "" && !strings.Contains(name, req.Info) {
			continue
		}
		v := dto.Volume{
			Name:       name,
			Driver:     strings.TrimSpace(fmt.Sprint(item["Driver"])),
			Mountpoint: strings.TrimSpace(fmt.Sprint(item["Mountpoint"])),
		}
		if labels, ok := item["Labels"].(map[string]interface{}); ok {
			for k, lv := range labels {
				v.Labels = append(v.Labels, fmt.Sprintf("%s=%v", k, lv))
			}
		}
		if created := strings.TrimSpace(fmt.Sprint(item["CreatedAt"])); created != "" {
			if t, e := time.Parse(time.RFC3339Nano, created); e == nil {
				v.CreatedAt = t
			}
		}
		list = append(list, v)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt.After(list[j].CreatedAt) })
	total := len(list)
	start, end := (req.Page-1)*req.Limit, req.Page*req.Limit
	if start < 0 {
		start = 0
	}
	if end > total {
		end = total
	}
	if start > total {
		return int64(total), []dto.Volume{}, nil
	}
	return int64(total), list[start:end], nil
}

func (u *ContainerService) listVolumePodman() ([]dto.Options, error) {
	if err := docker.PodmanEnsureReady(context.Background()); err != nil {
		return nil, err
	}
	cmdExec := exec.Command("podman", "volume", "ls", "--format", "json")
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
		datas = append(datas, dto.Options{Option: strings.TrimSpace(fmt.Sprint(item["Name"]))})
	}
	sort.Slice(datas, func(i, j int) bool { return datas[i].Option < datas[j].Option })
	return datas, nil
}

func (u *ContainerService) deleteVolumePodman(req *dto.BatchDelete) error {
	if err := docker.PodmanEnsureReady(context.Background()); err != nil {
		return err
	}
	for _, id := range req.Names {
		c := exec.Command("podman", "volume", "rm", "-f", id)
		out, err := c.CombinedOutput()
		if err != nil {
			msg := strings.TrimSpace(string(out))
			if strings.Contains(strings.ToLower(msg), "in use") {
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

func (u *ContainerService) createVolumePodman(req *dto.VolumeCreate) error {
	if err := docker.PodmanEnsureReady(context.Background()); err != nil {
		return err
	}
	check := exec.Command("podman", "volume", "exists", req.Name)
	if err := check.Run(); err == nil {
		return constant.ErrRecordExist
	}
	args := []string{"volume", "create", "--driver", req.Driver}
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

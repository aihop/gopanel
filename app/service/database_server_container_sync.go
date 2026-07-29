package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/utils/docker"
	containertypes "github.com/docker/docker/api/types"
	"gorm.io/gorm"
)

const containerDatabaseRemarkPrefix = "auto synced from running container: "

var containerDatabaseSyncMutex sync.Mutex

type ContainerDatabaseSyncItem struct {
	Container string             `json:"container"`
	Type      model.DatabaseType `json:"type"`
	Status    string             `json:"status"`
	Reason    string             `json:"reason,omitempty"`
}

type ContainerDatabaseSyncResult struct {
	Scanned  int                         `json:"scanned"`
	Detected int                         `json:"detected"`
	Created  int                         `json:"created"`
	Updated  int                         `json:"updated"`
	Skipped  int                         `json:"skipped"`
	Failed   int                         `json:"failed"`
	Items    []ContainerDatabaseSyncItem `json:"items"`
}

type containerDatabaseInspect struct {
	Name   string `json:"Name"`
	Config struct {
		Image string   `json:"Image"`
		Env   []string `json:"Env"`
	} `json:"Config"`
	HostConfig struct {
		NetworkMode string `json:"NetworkMode"`
	} `json:"HostConfig"`
	NetworkSettings struct {
		Ports map[string][]struct {
			HostIP   string `json:"HostIp"`
			HostPort string `json:"HostPort"`
		} `json:"Ports"`
	} `json:"NetworkSettings"`
}

type containerDatabaseCandidate struct {
	Name     string
	Type     model.DatabaseType
	Host     string
	Port     uint
	Username string
	Password string
}

func (s DatabaseServerService) SyncRunningContainers(parent context.Context) (ContainerDatabaseSyncResult, error) {
	containerDatabaseSyncMutex.Lock()
	defer containerDatabaseSyncMutex.Unlock()

	ctx, cancel := context.WithTimeout(parent, 45*time.Second)
	defer cancel()
	containers, sourceByID, err := getContainerListView(ctx)
	if err != nil {
		return ContainerDatabaseSyncResult{}, err
	}

	result := ContainerDatabaseSyncResult{Items: make([]ContainerDatabaseSyncItem, 0)}
	for _, item := range containers {
		if !strings.EqualFold(strings.TrimSpace(item.State), "running") {
			continue
		}
		result.Scanned++
		dbType, ok := databaseTypeFromContainerImage(item.Image)
		if !ok {
			continue
		}
		result.Detected++

		inspect, inspectErr := inspectContainerDatabase(ctx, item.ID, sourceByID[item.ID])
		if inspectErr != nil {
			result.addItem(containerPrimaryName(item), dbType, "failed", inspectErr.Error())
			continue
		}
		candidate, candidateErr := buildContainerDatabaseCandidate(item, inspect, dbType)
		if candidateErr != nil {
			result.addItem(containerPrimaryName(item), dbType, "skipped", candidateErr.Error())
			continue
		}
		status, syncErr := s.syncContainerDatabaseCandidate(candidate)
		if syncErr != nil {
			result.addItem(candidate.Name, candidate.Type, "failed", syncErr.Error())
			continue
		}
		result.addItem(candidate.Name, candidate.Type, status, "")
	}
	return result, nil
}

func (r *ContainerDatabaseSyncResult) addItem(container string, dbType model.DatabaseType, status, reason string) {
	switch status {
	case "created":
		r.Created++
	case "updated":
		r.Updated++
	case "skipped":
		r.Skipped++
	case "failed":
		r.Failed++
	}
	r.Items = append(r.Items, ContainerDatabaseSyncItem{Container: container, Type: dbType, Status: status, Reason: reason})
}

func inspectContainerDatabase(ctx context.Context, containerID, runtimeHost string) (containerDatabaseInspect, error) {
	cmd, err := docker.RuntimeCommandWithHost(ctx, runtimeHost, "inspect", containerID)
	if err != nil {
		return containerDatabaseInspect{}, err
	}
	output, err := cmd.Output()
	if err != nil {
		return containerDatabaseInspect{}, fmt.Errorf("inspect container: %w", err)
	}
	var inspections []containerDatabaseInspect
	if err := json.Unmarshal(output, &inspections); err != nil {
		return containerDatabaseInspect{}, fmt.Errorf("parse container inspect: %w", err)
	}
	if len(inspections) == 0 {
		return containerDatabaseInspect{}, errors.New("container inspect returned no data")
	}
	return inspections[0], nil
}

func databaseTypeFromContainerImage(image string) (model.DatabaseType, bool) {
	image = strings.ToLower(strings.TrimSpace(image))
	switch {
	case strings.Contains(image, "postgres"), strings.Contains(image, "postgis"):
		return model.DatabaseTypePostgresql, true
	case strings.Contains(image, "mysql"), strings.Contains(image, "mariadb"), strings.Contains(image, "percona"):
		return model.DatabaseTypeMysql, true
	default:
		return "", false
	}
}

func buildContainerDatabaseCandidate(item containertypes.Container, inspect containerDatabaseInspect, dbType model.DatabaseType) (containerDatabaseCandidate, error) {
	name := strings.TrimPrefix(strings.TrimSpace(inspect.Name), "/")
	if name == "" {
		name = containerPrimaryName(item)
	}
	env := parseContainerEnv(inspect.Config.Env)
	if !hasContainerDatabaseEnv(dbType, env) {
		return containerDatabaseCandidate{}, errors.New("database environment variables were not found")
	}
	username, password, privatePort := containerDatabaseCredentials(dbType, env)
	if username == "" || password == "" {
		return containerDatabaseCandidate{}, errors.New("database administrator credentials are incomplete")
	}
	host, port, ok := containerDatabasePublishedEndpoint(inspect, item.Ports, privatePort)
	if !ok {
		return containerDatabaseCandidate{}, fmt.Errorf("container port %d is not published to the host", privatePort)
	}
	return containerDatabaseCandidate{Name: name, Type: dbType, Host: host, Port: port, Username: username, Password: password}, nil
}

func parseContainerEnv(values []string) map[string]string {
	env := make(map[string]string, len(values))
	for _, value := range values {
		key, val, ok := strings.Cut(value, "=")
		if ok {
			env[strings.TrimSpace(key)] = val
		}
	}
	return env
}

func hasContainerDatabaseEnv(dbType model.DatabaseType, env map[string]string) bool {
	var keys []string
	switch dbType {
	case model.DatabaseTypeMysql:
		keys = []string{"PANEL_DB_ROOT_USER", "PANEL_DB_ROOT_PASSWORD", "MYSQL_ROOT_PASSWORD", "MARIADB_ROOT_PASSWORD", "MYSQL_USER", "MYSQL_PASSWORD", "MARIADB_USER", "MARIADB_PASSWORD"}
	case model.DatabaseTypePostgresql:
		keys = []string{"PANEL_DB_ROOT_USER", "PANEL_DB_ROOT_PASSWORD", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRESQL_USERNAME", "POSTGRESQL_POSTGRES_PASSWORD", "POSTGRESQL_PASSWORD"}
	}
	for _, key := range keys {
		if _, ok := env[key]; ok {
			return true
		}
	}
	return false
}

func containerDatabaseCredentials(dbType model.DatabaseType, env map[string]string) (string, string, uint) {
	first := func(keys ...string) string {
		for _, key := range keys {
			if value := strings.TrimSpace(env[key]); value != "" {
				return value
			}
		}
		return ""
	}
	switch dbType {
	case model.DatabaseTypeMysql:
		username := first("PANEL_DB_ROOT_USER", "MYSQL_ROOT_USER")
		if username == "" {
			username = "root"
		}
		if password := first("PANEL_DB_ROOT_PASSWORD", "MYSQL_ROOT_PASSWORD", "MARIADB_ROOT_PASSWORD"); password != "" {
			return username, password, 3306
		}
		return first("MYSQL_USER", "MARIADB_USER"), first("MYSQL_PASSWORD", "MARIADB_PASSWORD"), 3306
	case model.DatabaseTypePostgresql:
		username := first("PANEL_DB_ROOT_USER", "POSTGRES_USER", "POSTGRESQL_USERNAME")
		if username == "" {
			username = "postgres"
		}
		return username, first("PANEL_DB_ROOT_PASSWORD", "POSTGRES_PASSWORD", "POSTGRESQL_POSTGRES_PASSWORD", "POSTGRESQL_PASSWORD"), 5432
	default:
		return "", "", 0
	}
}

func containerDatabasePublishedEndpoint(inspect containerDatabaseInspect, listedPorts []containertypes.Port, privatePort uint) (string, uint, bool) {
	if strings.EqualFold(strings.TrimSpace(inspect.HostConfig.NetworkMode), "host") {
		return "127.0.0.1", privatePort, true
	}
	bindings := inspect.NetworkSettings.Ports[fmt.Sprintf("%d/tcp", privatePort)]
	for _, binding := range bindings {
		port, err := strconv.ParseUint(strings.TrimSpace(binding.HostPort), 10, 16)
		if err != nil || port == 0 {
			continue
		}
		host := normalizeContainerDatabaseHost(binding.HostIP)
		return host, uint(port), true
	}
	for _, binding := range listedPorts {
		if uint(binding.PrivatePort) != privatePort || binding.PublicPort == 0 || !strings.EqualFold(binding.Type, "tcp") {
			continue
		}
		host := normalizeContainerDatabaseHost(binding.IP)
		return host, uint(binding.PublicPort), true
	}
	return "", 0, false
}

func normalizeContainerDatabaseHost(host string) string {
	switch strings.TrimSpace(host) {
	case "", "0.0.0.0", "::", "::1":
		return "127.0.0.1"
	default:
		return strings.TrimSpace(host)
	}
}

func (s DatabaseServerService) syncContainerDatabaseCandidate(candidate containerDatabaseCandidate) (string, error) {
	existingByName, err := s.repo.GetByName(candidate.Name)
	if err == nil && existingByName.ID > 0 {
		if existingByName.Type != candidate.Type || !strings.HasPrefix(existingByName.Remark, containerDatabaseRemarkPrefix) {
			return "skipped", nil
		}
		if existing, endpointErr := s.repo.GetByEndpoint(candidate.Type, candidate.Host, candidate.Port); endpointErr == nil && existing.ID != existingByName.ID {
			return "skipped", nil
		} else if endpointErr != nil && !errors.Is(endpointErr, gorm.ErrRecordNotFound) {
			return "", endpointErr
		}
		if containerDatabaseServerUnchanged(existingByName, candidate) {
			return "skipped", nil
		}
		existingByName.Host = candidate.Host
		existingByName.Port = candidate.Port
		existingByName.Username = candidate.Username
		existingByName.Password = candidate.Password
		existingByName.Mode = model.DatabaseModeRemote
		existingByName.Remark = containerDatabaseRemarkPrefix + candidate.Name
		if !checkServer(&existingByName) {
			return "", errors.New("database connection check failed")
		}
		return "updated", s.repo.Update(&existingByName)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}
	if existing, endpointErr := s.repo.GetByEndpoint(candidate.Type, candidate.Host, candidate.Port); endpointErr == nil && existing.ID > 0 {
		return "skipped", nil
	} else if endpointErr != nil && !errors.Is(endpointErr, gorm.ErrRecordNotFound) {
		return "", endpointErr
	}

	server := &model.DatabaseServer{Name: candidate.Name, Type: candidate.Type, Host: candidate.Host, Port: candidate.Port, Username: candidate.Username, Password: candidate.Password, Mode: model.DatabaseModeRemote, Remark: containerDatabaseRemarkPrefix + candidate.Name}
	if !checkServer(server) {
		return "", errors.New("database connection check failed")
	}
	if err := s.repo.Create(server); err != nil {
		return "", err
	}
	return "created", nil
}

func containerDatabaseServerUnchanged(server model.DatabaseServer, candidate containerDatabaseCandidate) bool {
	return server.Type == candidate.Type &&
		server.Host == candidate.Host &&
		server.Port == candidate.Port &&
		server.Username == candidate.Username &&
		server.Password == candidate.Password &&
		server.Mode == model.DatabaseModeRemote &&
		server.Remark == containerDatabaseRemarkPrefix+candidate.Name
}

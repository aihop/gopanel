package service

import (
	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/pkg/websocket"
	"github.com/docker/docker/api/types"
	"sync"
	"time"
)

type ContainerService struct{}

const (
	containerListViewCacheTTL  = 3 * time.Second
	containerListStatsCacheTTL = 2 * time.Second
)

type containerListViewCacheEntry struct {
	expireAt time.Time
	items    []types.Container
	source   map[string]string
}
type containerListStatsCacheEntry struct {
	expireAt time.Time
	items    []dto.ContainerListStats
}

var (
	containerListViewCache struct {
		mu         sync.RWMutex
		entry      containerListViewCacheEntry
		refreshing bool
		waitCh     chan struct{}
		version    uint64
	}
	containerListStatsCache struct {
		mu    sync.RWMutex
		entry containerListStatsCacheEntry
	}
)

type IContainerService interface {
	Page(req *dto.PageContainer) (int64, interface{}, error)
	List() ([]string, error)
	PageNetwork(req *dto.SearchWithPage) (int64, interface{}, error)
	ListNetwork() ([]dto.Options, error)
	PageVolume(req *dto.SearchWithPage) (int64, interface{}, error)
	ListVolume() ([]dto.Options, error)
	PageCompose(req *dto.SearchWithPage) (int64, interface{}, error)
	CreateCompose(req *dto.ComposeCreate) (string, error)
	ComposeOperation(req *dto.ComposeOperation) error
	ContainerCreate(req *dto.ContainerOperate) error
	ContainerUpdate(req *dto.ContainerOperate) error
	ContainerUpgrade(req *dto.ContainerUpgrade) error
	ContainerInfo(req *dto.OperationWithName) (*dto.ContainerOperate, error)
	ContainerListStats() ([]dto.ContainerListStats, error)
	LoadResourceLimit() (*dto.ResourceLimit, error)
	ContainerRename(req *dto.ContainerRename) error
	ContainerCommit(req *dto.ContainerCommit) error
	ContainerLogClean(req *dto.OperationWithName) error
	ContainerOperation(req *dto.ContainerOperation) error
	ContainerLogs(wsConn *websocket.Conn, containerType, container, since, tail, runtimeHost string, follow bool) error
	DownloadContainerLogs(containerType, container, since, tail, runtimeHost string) (string, error)
	ContainerStatsByID(id string) (*dto.ContainerStats, error)
	Inspect(req *dto.InspectReq) (string, error)
	DeleteNetwork(req *dto.BatchDelete) error
	CreateNetwork(req *dto.NetworkCreate) error
	DeleteVolume(req *dto.BatchDelete) error
	CreateVolume(req *dto.VolumeCreate) error
	TestCompose(req *dto.ComposeCreate) (bool, error)
	ComposeUpdate(req *dto.ComposeUpdate) error
	Prune(req *dto.ContainerPrune) (dto.ContainerPruneReport, error)
	LoadContainerLogs(req *dto.OperationWithNameAndType) string
}

func NewIContainerService() IContainerService {
	return &ContainerService{}
}

type containerPageRelatedMeta struct {
	installByContainerName    map[string]model.AppInstall
	websiteDomainsByInstallID map[uint][]string
	websiteByContainerID      map[string]model.Website
}

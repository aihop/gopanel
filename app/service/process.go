package service

import (
	"sort"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/dto/response"
	"github.com/aihop/gopanel/utils/common"
	"github.com/shirou/gopsutil/v4/process"
)

type ProcessService struct{}

const processInitialListLimit = 20

type IProcessService interface {
	List(req request.ProcessListReq) ([]response.ProcessListItem, error)
	StopProcess(req request.ProcessReq) error
	CheckProcessPort(port uint32) (int32, error)
}

func NewIProcessService() IProcessService {
	return &ProcessService{}
}

func (p *ProcessService) List(req request.ProcessListReq) ([]response.ProcessListItem, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, err
	}

	result := make([]response.ProcessListItem, 0, len(processes))
	for _, proc := range processes {
		item := response.ProcessListItem{
			PID:            proc.Pid,
			CpuPercent:     "--",
			DiskRead:       "--",
			DiskWrite:      "--",
			NumConnections: 0,
			Envs:           []string{},
			OpenFiles:      []response.ProcessOpenFile{},
			Connects:       []response.ProcessConnection{},
		}
		if req.PID > 0 && req.PID != proc.Pid {
			continue
		}
		if name, err := proc.Name(); err == nil {
			item.Name = name
		} else {
			item.Name = "<UNKNOWN>"
		}
		if req.Name != "" && !strings.Contains(item.Name, req.Name) {
			continue
		}
		if username, err := proc.Username(); err == nil {
			item.Username = username
		}
		if req.Username != "" && !strings.Contains(item.Username, req.Username) {
			continue
		}
		item.PPID, _ = proc.Ppid()
		statusArray, _ := proc.Status()
		if len(statusArray) > 0 {
			item.Status = strings.Join(statusArray, ",")
		}
		if createTime, err := proc.CreateTime(); err == nil {
			item.StartTime = time.Unix(createTime/1000, 0).Format("2006-1-2 15:04:05")
		}
		item.NumThreads, _ = proc.NumThreads()
		if memInfo, err := proc.MemoryInfo(); err == nil {
			item.Rss = common.FormatBytes(memInfo.RSS)
			item.RssValue = memInfo.RSS
			item.Data = common.FormatBytes(memInfo.Data)
			item.VMS = common.FormatBytes(memInfo.VMS)
			item.HWM = common.FormatBytes(memInfo.HWM)
			item.Stack = common.FormatBytes(memInfo.Stack)
			item.Locked = common.FormatBytes(memInfo.Locked)
			item.Swap = common.FormatBytes(memInfo.Swap)
		} else {
			item.Rss = "--"
			item.Data = "--"
			item.VMS = "--"
			item.HWM = "--"
			item.Stack = "--"
			item.Locked = "--"
			item.Swap = "--"
		}
		result = append(result, item)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].PID < result[j].PID
	})
	if len(result) > processInitialListLimit {
		result = result[:processInitialListLimit]
	}
	return result, nil
}

func (p *ProcessService) StopProcess(req request.ProcessReq) error {
	proc, err := process.NewProcess(req.PID)
	if err != nil {
		return err
	}
	if err := proc.Kill(); err != nil {
		return err
	}
	return nil
}

// 检查端口占用，如果被占用，返回占用的进程pid
func (p *ProcessService) CheckProcessPort(port uint32) (int32, error) {
	procList, err := process.Processes()
	if err != nil {
		return 0, err
	}
	for _, proc := range procList {
		connList, err := proc.Connections()
		if err != nil {
			return 0, err
		}
		for _, conn := range connList {
			if conn.Laddr.Port == port {
				return conn.Pid, nil
				// return fmt.Errorf("port %d is occupied by process %d", conn.Laddr.Port, conn.Pid)
			}
		}
	}
	return 0, nil
}

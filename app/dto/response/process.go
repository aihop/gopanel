package response

type ProcessOpenFile struct {
	Path string `json:"path"`
	Fd   uint64 `json:"fd"`
}

type ProcessAddress struct {
	IP   string `json:"ip"`
	Port uint32 `json:"port"`
}

type ProcessConnection struct {
	Type       string         `json:"type"`
	Status     string         `json:"status"`
	LocalAddr  ProcessAddress `json:"localaddr"`
	RemoteAddr ProcessAddress `json:"remoteaddr"`
	PID        int32          `json:"PID"`
	Name       string         `json:"name"`
}

type ProcessListItem struct {
	PID            int32               `json:"PID"`
	Name           string              `json:"name"`
	PPID           int32               `json:"PPID"`
	Username       string              `json:"username"`
	Status         string              `json:"status"`
	StartTime      string              `json:"startTime"`
	NumThreads     int32               `json:"numThreads"`
	NumConnections int                 `json:"numConnections"`
	CpuPercent     string              `json:"cpuPercent"`
	DiskRead       string              `json:"diskRead"`
	DiskWrite      string              `json:"diskWrite"`
	CmdLine        string              `json:"cmdLine"`
	Rss            string              `json:"rss"`
	VMS            string              `json:"vms"`
	HWM            string              `json:"hwm"`
	Data           string              `json:"data"`
	Stack          string              `json:"stack"`
	Locked         string              `json:"locked"`
	Swap           string              `json:"swap"`
	CpuValue       float64             `json:"cpuValue"`
	RssValue       uint64              `json:"rssValue"`
	Envs           []string            `json:"envs"`
	OpenFiles      []ProcessOpenFile   `json:"openFiles"`
	Connects       []ProcessConnection `json:"connects"`
}

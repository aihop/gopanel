package service

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/common"
)

func checkPortUsed(ports, proto string, apps []portOfApp) string {
	var portList []int
	if strings.Contains(ports, "-") || strings.Contains(ports, ",") {
		if strings.Contains(ports, "-") {
			port1, err := strconv.Atoi(strings.Split(ports, "-")[0])
			if err != nil {
				global.LOG.Errorf(" convert string %s to int failed, err: %v", strings.Split(ports, "-")[0], err)
				return ""
			}
			port2, err := strconv.Atoi(strings.Split(ports, "-")[1])
			if err != nil {
				global.LOG.Errorf(" convert string %s to int failed, err: %v", strings.Split(ports, "-")[1], err)
				return ""
			}
			for i := port1; i <= port2; i++ {
				portList = append(portList, i)
			}
		} else {
			portLists := strings.Split(ports, ",")
			for _, item := range portLists {
				portItem, _ := strconv.Atoi(item)
				portList = append(portList, portItem)
			}
		}

		var usedPorts []string
		for _, port := range portList {
			portItem := fmt.Sprintf("%v", port)
			isUsedByApp := false
			for _, app := range apps {
				if app.HttpPort == portItem || app.HttpsPort == portItem {
					isUsedByApp = true
					usedPorts = append(usedPorts, fmt.Sprintf("%s (%s)", portItem, app.AppName))
					break
				}
			}
			if !isUsedByApp && common.ScanPortWithProto(port, proto) {
				usedPorts = append(usedPorts, fmt.Sprintf("%v", port))
			}
		}
		return strings.Join(usedPorts, ",")
	}

	for _, app := range apps {
		if app.HttpPort == ports || app.HttpsPort == ports {
			return fmt.Sprintf("(%s)", app.AppName)
		}
	}
	port, err := strconv.Atoi(ports)
	if err != nil {
		global.LOG.Errorf(" convert string %v to int failed, err: %v", port, err)
		return ""
	}
	if common.ScanPortWithProto(port, proto) {
		return "inUsed"
	}
	return ""
}

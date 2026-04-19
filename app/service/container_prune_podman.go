package service

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/utils/docker"
)

var pruneHexRe = regexp.MustCompile(`^[0-9a-f]{6,}$`)

func (u *ContainerService) prunePodman(req *dto.ContainerPrune) (dto.ContainerPruneReport, error) {
	report := dto.ContainerPruneReport{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	args := []string{}
	switch req.PruneType {
	case "container":
		args = append(args, "container", "prune", "-f")
	case "image":
		args = append(args, "image", "prune", "-f")
		if req.WithTagAll {
			args = append(args, "-a")
		}
	case "network":
		args = append(args, "network", "prune", "-f")
	case "volume":
		args = append(args, "volume", "prune", "-f")
	case "buildcache":
		args = append(args, "builder", "prune", "-f")
		if req.WithTagAll {
			args = append(args, "-a")
		}
	default:
		return report, errors.New("unknown prune type")
	}

	if req.WithTagAll && req.PruneType != "image" {
		args = append(args, "--filter", "until=24h")
	}

	c, err := docker.RuntimeCommand(ctx, args...)
	if err != nil {
		return report, err
	}
	out, runErr := c.CombinedOutput()
	msg := string(out)
	if runErr != nil {
		m := strings.TrimSpace(msg)
		if m == "" {
			return report, runErr
		}
		return report, errors.New(m)
	}

	deleted, reclaimed := parsePodmanPruneOutput(msg)
	report.DeletedNumber = deleted
	report.SpaceReclaimed = reclaimed
	return report, nil
}

func parsePodmanPruneOutput(out string) (int, int) {
	deleted := 0
	reclaimed := 0
	lines := strings.Split(out, "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "total reclaimed space") {
			if n, ok := parseReclaimedBytes(line); ok {
				reclaimed = n
			}
			continue
		}
		if strings.HasPrefix(lower, "deleted:") || strings.HasPrefix(lower, "untagged:") {
			deleted++
			continue
		}
		if pruneHexRe.MatchString(strings.ToLower(line)) {
			deleted++
			continue
		}
	}
	return deleted, reclaimed
}

func parseReclaimedBytes(line string) (int, bool) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return 0, false
	}
	v := strings.TrimSpace(parts[1])
	if v == "" {
		return 0, false
	}
	v = strings.Fields(v)[0]
	numPart := ""
	unitPart := ""
	for i := 0; i < len(v); i++ {
		ch := v[i]
		if (ch >= '0' && ch <= '9') || ch == '.' {
			numPart += string(ch)
		} else {
			unitPart = v[i:]
			break
		}
	}
	if numPart == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(numPart, 64)
	if err != nil {
		return 0, false
	}
	unit := strings.ToLower(strings.TrimSpace(unitPart))
	m := 1.0
	switch unit {
	case "b", "":
		m = 1
	case "kb", "kib":
		m = 1024
	case "mb", "mib":
		m = 1024 * 1024
	case "gb", "gib":
		m = 1024 * 1024 * 1024
	case "tb", "tib":
		m = 1024 * 1024 * 1024 * 1024
	default:
		if strings.HasSuffix(unit, "b") {
			u := strings.TrimSuffix(unit, "b")
			switch u {
			case "k", "ki":
				m = 1024
			case "m", "mi":
				m = 1024 * 1024
			case "g", "gi":
				m = 1024 * 1024 * 1024
			case "t", "ti":
				m = 1024 * 1024 * 1024 * 1024
			default:
				return 0, false
			}
		} else {
			return 0, false
		}
	}
	n := int(f * m)
	if n < 0 {
		return 0, false
	}
	return n, true
}

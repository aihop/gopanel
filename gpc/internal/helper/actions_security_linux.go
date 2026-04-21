package helper

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

func (s *Server) actionSecurityScanSSH(ctx context.Context, params map[string]interface{}) (string, error) {
	configPath := "/etc/ssh/sshd_config"
	content, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read sshd_config: %w", err)
	}

	configStr := string(content)
	lines := strings.Split(configStr, "\n")

	result := map[string]interface{}{
		"port":                   22,
		"permitRootLogin":        "yes",
		"passwordAuthentication": "yes",
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 2 {
			key := strings.ToLower(parts[0])
			val := strings.ToLower(parts[1])

			switch key {
			case "port":
				result["port"] = val
			case "permitrootlogin":
				result["permitRootLogin"] = val
			case "passwordauthentication":
				result["passwordAuthentication"] = val
			}
		}
	}

	resBytes, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(resBytes), nil
}

func (s *Server) actionSecurityFixSSH(ctx context.Context, params map[string]interface{}) (string, error) {
	configPath := "/etc/ssh/sshd_config"
	content, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read sshd_config: %w", err)
	}

	configStr := string(content)
	lines := strings.Split(configStr, "\n")

	var newLines []string
	portFixed := false
	rootFixed := false
	pwdFixed := false

	// Generate a random high port between 30000 and 60000
	rand.Seed(time.Now().UnixNano())
	newPort := rand.Intn(30000) + 30000

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") || trimmed == "" {
			newLines = append(newLines, line)
			continue
		}

		parts := strings.Fields(trimmed)
		if len(parts) >= 2 {
			key := strings.ToLower(parts[0])

			switch key {
			case "port":
				newLines = append(newLines, fmt.Sprintf("Port %d", newPort))
				portFixed = true
			case "permitrootlogin":
				newLines = append(newLines, "PermitRootLogin no")
				rootFixed = true
			case "passwordauthentication":
				newLines = append(newLines, "PasswordAuthentication no")
				pwdFixed = true
			default:
				newLines = append(newLines, line)
			}
		} else {
			newLines = append(newLines, line)
		}
	}

	if !portFixed {
		newLines = append(newLines, fmt.Sprintf("Port %d", newPort))
	}
	if !rootFixed {
		newLines = append(newLines, "PermitRootLogin no")
	}
	if !pwdFixed {
		newLines = append(newLines, "PasswordAuthentication no")
	}

	newContent := strings.Join(newLines, "\n")
	if err := os.WriteFile(configPath, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write sshd_config: %w", err)
	}

	// Restart SSHD
	cmd := exec.CommandContext(ctx, "systemctl", "restart", "sshd")
	if out, err := cmd.CombinedOutput(); err != nil {
		// Fallback to ssh
		cmd = exec.CommandContext(ctx, "systemctl", "restart", "ssh")
		if out2, err2 := cmd.CombinedOutput(); err2 != nil {
			return "", fmt.Errorf("failed to restart sshd: %w, output: %s, %s", err2, string(out), string(out2))
		}
	}

	result := map[string]interface{}{
		"newPort": newPort,
		"msg":     "SSH 加固完成，请切记使用新端口和密钥登录。",
	}
	resBytes, _ := json.Marshal(result)
	return string(resBytes), nil
}

func (s *Server) actionSecurityScanPort(ctx context.Context, params map[string]interface{}) (string, error) {
	// Look for typical database ports open to 0.0.0.0
	// For example: 3306 (MySQL), 6379 (Redis), 5432 (PostgreSQL), 27017 (MongoDB)
	highRiskPorts := []string{"3306", "6379", "5432", "27017"}
	
	cmd := exec.CommandContext(ctx, "ss", "-tulnp")
	out, err := cmd.CombinedOutput()
	if err != nil {
		// fallback to netstat
		cmd = exec.CommandContext(ctx, "netstat", "-tulnp")
		out, err = cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("failed to execute ss or netstat: %w", err)
		}
	}

	var exposedPorts []string
	outStr := string(out)
	lines := strings.Split(outStr, "\n")
	
	// A simple regex to find 0.0.0.0:port or :::port
	re := regexp.MustCompile(`(0\.0\.0\.0|:::|::|127\.0\.0\.1|localhost):(\d+)`)

	foundPorts := make(map[string]bool)
	for _, line := range lines {
		if !strings.Contains(line, "LISTEN") {
			continue
		}
		
		matches := re.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) == 3 {
				ip := match[1]
				port := match[2]
				
				// We only care about exposed ports
				if ip == "0.0.0.0" || ip == ":::" || ip == "::" {
					for _, riskPort := range highRiskPorts {
						if port == riskPort && !foundPorts[riskPort] {
							exposedPorts = append(exposedPorts, riskPort)
							foundPorts[riskPort] = true
						}
					}
				}
			}
		}
	}

	resBytes, _ := json.Marshal(map[string]interface{}{
		"exposed": exposedPorts,
	})
	return string(resBytes), nil
}

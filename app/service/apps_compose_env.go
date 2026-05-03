package service

import (
	"fmt"
	"github.com/aihop/gopanel/utils/docker"
	"gopkg.in/yaml.v3"
	"sort"
	"strings"
)

func ensureExternalNetworks(composeYml string) error {
	composeMap := make(map[string]interface{})
	if err := yaml.Unmarshal([]byte(composeYml), &composeMap); err != nil {
		return err
	}
	netsAny, ok := composeMap["networks"]
	if !ok || netsAny == nil {
		return nil
	}
	nets, ok := netsAny.(map[string]interface{})
	if !ok {
		return nil
	}
	for k, v := range nets {
		name := strings.TrimSpace(k)
		external := false
		if m, ok := v.(map[string]interface{}); ok {
			if ex, ok := m["external"]; ok {
				switch x := ex.(type) {
				case bool:
					external = x
				case map[string]interface{}:
					external = true
					if n, ok := x["name"]; ok {
						if s := strings.TrimSpace(fmt.Sprint(n)); s != "" {
							name = s
						}
					}
				default:
					if strings.EqualFold(strings.TrimSpace(fmt.Sprint(ex)), "true") {
						external = true
					}
				}
			}
			if external {
				if exName, ok := m["name"]; ok {
					if s := strings.TrimSpace(fmt.Sprint(exName)); s != "" {
						name = s
					}
				}
			}
		}
		if !external {
			continue
		}
		if err := docker.EnsureNetwork(name); err != nil {
			return fmt.Errorf("external network %s not available: %w", name, err)
		}
	}
	return nil
}
func validateComposeEnvForPortsVolumes(composeYml string, envText string) error {
	envMap := parseDotEnv(envText)
	required := extractRequiredVarsFromComposePortsVolumes(composeYml)
	if len(required) == 0 {
		return nil
	}
	var missing []string
	for _, k := range required {
		v := strings.TrimSpace(envMap[k])
		v = strings.Trim(v, `"`)
		if v == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	return fmt.Errorf("missing required compose variables (ports/volumes): %s", strings.Join(missing, ", "))
}
func parseDotEnv(envText string) map[string]string {
	out := make(map[string]string)
	lines := strings.Split(strings.ReplaceAll(envText, "\r\n", "\n"), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		out[k] = v
	}
	return out
}
func normalizeRedisPasswordEnvAliases(envMap map[string]string) bool {
	rootPassword := strings.TrimSpace(envMap["PANEL_REDIS_ROOT_PASSWORD"])
	password := strings.TrimSpace(envMap["PANEL_REDIS_PASSWORD"])
	changed := false
	if rootPassword != "" && password == "" {
		envMap["PANEL_REDIS_PASSWORD"] = rootPassword
		changed = true
	}
	if password != "" && rootPassword == "" {
		envMap["PANEL_REDIS_ROOT_PASSWORD"] = password
		changed = true
	}
	return changed
}
func normalizeRedisPasswordParamAliases(params map[string]interface{}) bool {
	if params == nil {
		return false
	}
	rootPassword := strings.TrimSpace(fmt.Sprint(params["PANEL_REDIS_ROOT_PASSWORD"]))
	if rootPassword == "<nil>" {
		rootPassword = ""
	}
	password := strings.TrimSpace(fmt.Sprint(params["PANEL_REDIS_PASSWORD"]))
	if password == "<nil>" {
		password = ""
	}
	changed := false
	if rootPassword != "" && password == "" {
		params["PANEL_REDIS_PASSWORD"] = rootPassword
		changed = true
	}
	if password != "" && rootPassword == "" {
		params["PANEL_REDIS_ROOT_PASSWORD"] = password
		changed = true
	}
	return changed
}
func extractRequiredVarsFromComposePortsVolumes(composeYml string) []string {
	lines := strings.Split(strings.ReplaceAll(composeYml, "\r\n", "\n"), "\n")
	var (
		inPorts   bool
		inVolumes bool
		portsInd  int
		volInd    int
	)
	requiredSet := make(map[string]struct{})
	for _, line := range lines {
		raw := line
		line = strings.TrimRight(line, " \t")
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		ind := len(raw) - len(strings.TrimLeft(raw, " \t"))
		if inPorts && ind <= portsInd {
			inPorts = false
		}
		if inVolumes && ind <= volInd {
			inVolumes = false
		}
		if strings.HasSuffix(trim, "ports:") {
			inPorts = true
			portsInd = ind
			continue
		}
		if strings.HasSuffix(trim, "volumes:") {
			inVolumes = true
			volInd = ind
			continue
		}
		if !(inPorts || inVolumes) {
			continue
		}
		for _, m := range composeVarRe.FindAllStringSubmatch(line, -1) {
			if len(m) < 2 {
				continue
			}
			name, required := parseComposeVarExpr(m[1])
			if !required || name == "" {
				continue
			}
			requiredSet[name] = struct{}{}
		}
	}
	var required []string
	for k := range requiredSet {
		required = append(required, k)
	}
	sort.Strings(required)
	return required
}
func parseComposeVarExpr(expr string) (string, bool) {
	s := strings.TrimSpace(expr)
	if s == "" {
		return "", false
	}
	seps := []string{":-", ":-", ":+", ":?", "-", "+", "?"}
	name := s
	op := ""
	for _, sep := range seps {
		if i := strings.Index(s, sep); i > 0 {
			name = strings.TrimSpace(s[:i])
			op = sep
			break
		}
	}
	if name == "" {
		return "", false
	}
	required := op == "" || op == ":?" || op == "?"
	if strings.Contains(name, " ") || strings.Contains(name, "\t") {
		return "", false
	}
	return name, required
}

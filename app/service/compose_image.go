package service

import (
	"bufio"
	"bytes"
	"strings"

	"gopkg.in/yaml.v3"
)

func qualifyComposeImagesInMap(composeMap map[string]interface{}) {
	if composeMap == nil {
		return
	}
	servicesVal, ok := composeMap["services"]
	if !ok || servicesVal == nil {
		return
	}
	servicesMap, ok := servicesVal.(map[string]interface{})
	if !ok || len(servicesMap) == 0 {
		return
	}
	for _, v := range servicesMap {
		svc, ok := v.(map[string]interface{})
		if !ok || len(svc) == 0 {
			continue
		}
		raw, ok := svc["image"]
		if !ok || raw == nil {
			continue
		}
		img, ok := raw.(string)
		if !ok {
			continue
		}
		img = strings.TrimSpace(img)
		if img == "" {
			continue
		}
		if strings.HasPrefix(img, "${") {
			continue
		}
		newImg := qualifyImageName(img)
		if newImg != "" && newImg != img {
			svc["image"] = newImg
		}
	}
}

func qualifyComposeImagesYAML(composeYml string) (string, bool, error) {
	composeMap := make(map[string]interface{})
	if err := yaml.Unmarshal([]byte(composeYml), &composeMap); err != nil {
		return "", false, err
	}
	before := strings.TrimSpace(composeYml)
	qualifyComposeImagesInMap(composeMap)
	out, err := yaml.Marshal(composeMap)
	if err != nil {
		return "", false, err
	}
	after := strings.TrimSpace(string(out))
	if before == after {
		return composeYml, false, nil
	}
	return string(out), true, nil
}

func qualifyComposeImagesText(composeYml string) (string, bool) {
	var out bytes.Buffer
	sc := bufio.NewScanner(strings.NewReader(composeYml))
	changed := false
	for sc.Scan() {
		line := sc.Text()
		ltrim := strings.TrimSpace(line)
		if !strings.HasPrefix(ltrim, "image:") {
			out.WriteString(line)
			out.WriteByte('\n')
			continue
		}
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		rest := strings.TrimSpace(strings.TrimPrefix(ltrim, "image:"))
		if rest == "" || strings.HasPrefix(rest, "${") {
			out.WriteString(line)
			out.WriteByte('\n')
			continue
		}
		quote := byte(0)
		if len(rest) >= 2 {
			if (rest[0] == '"' && rest[len(rest)-1] == '"') || (rest[0] == '\'' && rest[len(rest)-1] == '\'') {
				quote = rest[0]
				rest = rest[1 : len(rest)-1]
			}
		}
		newVal := qualifyImageName(strings.TrimSpace(rest))
		if newVal != "" && newVal != rest {
			changed = true
			rest = newVal
		}
		if quote != 0 {
			rest = string(quote) + rest + string(quote)
		}
		out.WriteString(indent)
		out.WriteString("image: ")
		out.WriteString(rest)
		out.WriteByte('\n')
	}
	if err := sc.Err(); err != nil {
		return composeYml, false
	}
	if changed {
		return out.String(), true
	}
	return composeYml, false
}

func qualifyEnvImageVars(envContent string) (string, bool) {
	var out bytes.Buffer
	sc := bufio.NewScanner(strings.NewReader(envContent))
	changed := false
	for sc.Scan() {
		line := sc.Text()
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") || !strings.Contains(line, "=") {
			out.WriteString(line)
			out.WriteByte('\n')
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])
		valRaw := parts[1]
		val := strings.TrimSpace(valRaw)
		lk := strings.ToUpper(key)
		if lk == "IMAGE" || strings.HasSuffix(lk, "_IMAGE") {
			quote := byte(0)
			if len(val) >= 2 {
				if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
					quote = val[0]
					val = val[1 : len(val)-1]
				}
			}
			newVal := qualifyImageName(val)
			if newVal != "" && newVal != val {
				changed = true
				val = newVal
			}
			if quote != 0 {
				val = string(quote) + val + string(quote)
			}
			out.WriteString(key)
			out.WriteByte('=')
			out.WriteString(val)
			out.WriteByte('\n')
			continue
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	if err := sc.Err(); err != nil {
		return envContent, false
	}
	if changed {
		return out.String(), true
	}
	return envContent, false
}

func qualifyImageName(img string) string {
	if img == "" {
		return ""
	}
	if strings.HasPrefix(img, "docker.io/") || strings.HasPrefix(img, "quay.io/") || strings.HasPrefix(img, "ghcr.io/") {
		return img
	}
	first := img
	rest := ""
	if i := strings.Index(img, "/"); i >= 0 {
		first = img[:i]
		rest = img[i+1:]
	}
	if first == "localhost" || strings.Contains(first, ".") || strings.Contains(first, ":") {
		return img
	}
	if rest == "" {
		return "docker.io/library/" + img
	}
	return "docker.io/" + img
}

package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/utils/gpagent"
	"github.com/gofiber/fiber/v3"
)

type CaddyReq struct {
	PrimaryDomain string `json:"primaryDomain"`
	OtherDomains  string `json:"otherDomains"`
	Content       string `json:"content"`
}

func HttpDefaultList(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cf, err := getCaddyfile(ctx)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if cf == "" {
		return c.JSON(e.Succ())
	}
	return c.JSON(e.Succ(cf))
}

func HttpDefaultGet(c fiber.Ctx) error {
	req, err := e.BodyToStruct[CaddyReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cf, err := getCaddyfile(ctx)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if cf == "" {
		return c.JSON(e.Succ())
	}
	domains := domainsFromReq(req.PrimaryDomain, req.OtherDomains)
	if len(domains) == 0 {
		return c.JSON(e.Fail(errors.New("primaryDomain/otherDomains cannot be empty")))
	}
	blocks := extractCaddyBlocks(cf, domains)
	return c.JSON(e.Succ(blocks))
}

func HttpDefaultDelete(c fiber.Ctx) error {
	req, err := e.BodyToStruct[CaddyReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	domains := domainsFromReq(req.PrimaryDomain, req.OtherDomains)
	if len(domains) == 0 {
		return c.JSON(e.Fail(errors.New("primaryDomain/otherDomains cannot be empty")))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cf, err := getCaddyfile(ctx)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	next, changed := deleteCaddyBlocks(cf, domains)
	if !changed {
		return c.JSON(e.Succ(fiber.Map{"deleted": false}))
	}
	if err := applyCaddyfile(ctx, next); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"deleted": true}))
}

func HttpDefaultCheck(c fiber.Ctx) error {
	type CheckUrlReq struct {
		Domain string `json:"domain"`
	}
	req, err := e.BodyToStruct[CheckUrlReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if strings.TrimSpace(req.Domain) == "" {
		return c.JSON(e.Fail(errors.New("domain cannot be empty")))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cf, err := getCaddyfile(ctx)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	exist := caddyDomainExists(cf, req.Domain)
	return c.JSON(e.Succ(fiber.Map{
		"exist": exist,
	}))
}

func HttpDefaultRead(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cf, err := getCaddyfile(ctx)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(cf))
}

func HttpDefaultUpdate(c fiber.Ctx) error {
	req, err := e.BodyToStruct[CaddyReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if req.Content == "" {
		return c.JSON(e.Fail(fmt.Errorf("content cannot be empty")))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if strings.TrimSpace(req.PrimaryDomain) == "" {
		if err := applyCaddyfile(ctx, req.Content); err != nil {
			return c.JSON(e.Fail(err))
		}
		return c.JSON(e.Succ())
	}
	cf, err := getCaddyfile(ctx)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	domains := domainsFromReq(req.PrimaryDomain, req.OtherDomains)
	next, changed := replaceCaddyBlocks(cf, domains, req.Content)
	if !changed {
		return c.JSON(e.Succ(fiber.Map{"updated": false}))
	}
	if err := applyCaddyfile(ctx, next); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"updated": true}))
}

func HttpDefaultRestart(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cf, err := getCaddyfile(ctx)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if cf == "" {
		return c.JSON(e.Fail(errors.New("caddyfile is empty")))
	}
	if err := applyCaddyfile(ctx, cf); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func HttpDefaultStop(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := gpagent.Do(ctx, "CADDY_STOP", nil); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func HttpDefaultStatus(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := gpagent.Do(ctx, "CADDY_STATUS", nil)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var out any
	if err := json.Unmarshal([]byte(resp.Output), &out); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(out))
}

func HttpDefaultResolve(c fiber.Ctx) error {
	type ResolveReq struct {
		Domain       string `json:"domain"`
		Proxy        string `json:"proxy"`
		OtherDomains string `json:"otherDomains"`
	}
	req, err := e.BodyToStruct[ResolveReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if strings.TrimSpace(req.Domain) == "" || strings.TrimSpace(req.Proxy) == "" {
		return c.JSON(e.Fail(errors.New("domain/proxy cannot be empty")))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cf, err := getCaddyfile(ctx)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	next := appendResolveBlock(cf, req.Domain, req.Proxy, req.OtherDomains)
	if err := applyCaddyfile(ctx, next); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{
		"added": true,
	}))
}

type caddyGetConfigResp struct {
	Caddyfile string `json:"caddyfile"`
}

func getCaddyfile(ctx context.Context) (string, error) {
	resp, err := gpagent.Do(ctx, "CADDY_GET_CONFIG", nil)
	if err != nil {
		return "", err
	}
	var out caddyGetConfigResp
	if err := json.Unmarshal([]byte(resp.Output), &out); err != nil {
		return "", err
	}
	return out.Caddyfile, nil
}

func applyCaddyfile(ctx context.Context, content string) error {
	_, err := gpagent.Do(ctx, "CADDY_APPLY", map[string]interface{}{"caddyfile": content})
	return err
}

func domainsFromReq(primary, other string) []string {
	var out []string
	primary = strings.TrimSpace(primary)
	if primary != "" {
		out = append(out, primary)
	}
	other = strings.ReplaceAll(other, ",", "\n")
	for _, ln := range strings.Split(other, "\n") {
		d := strings.TrimSpace(ln)
		if d == "" {
			continue
		}
		out = append(out, d)
	}
	seen := make(map[string]struct{}, len(out))
	var uniq []string
	for _, d := range out {
		k := normalizeCaddyHost(d)
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		uniq = append(uniq, d)
	}
	return uniq
}

func normalizeCaddyHost(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimPrefix(s, "https://")
	return strings.TrimSpace(s)
}

func caddyDomainExists(content, domain string) bool {
	target := normalizeCaddyHost(domain)
	if target == "" {
		return false
	}
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		if !strings.HasSuffix(t, "{") {
			continue
		}
		header := strings.TrimSpace(strings.TrimSuffix(t, "{"))
		for _, p := range strings.Split(header, ",") {
			if normalizeCaddyHost(strings.TrimSpace(p)) == target {
				return true
			}
		}
	}
	return false
}

func extractCaddyBlocks(content string, domains []string) string {
	var blocks []string
	remain := content
	for _, d := range domains {
		b, _ := splitFirstBlock(remain, d)
		if b != "" {
			blocks = append(blocks, b)
		}
	}
	return strings.Join(blocks, "\n")
}

func replaceCaddyBlocks(content string, domains []string, replacement string) (string, bool) {
	next, changed := deleteCaddyBlocks(content, domains)
	if !changed {
		return content, false
	}
	next = strings.TrimRight(next, "\n") + "\n\n" + strings.TrimSpace(replacement) + "\n"
	return next, true
}

func deleteCaddyBlocks(content string, domains []string) (string, bool) {
	targets := make(map[string]struct{}, len(domains))
	for _, d := range domains {
		n := normalizeCaddyHost(d)
		if n == "" {
			continue
		}
		targets[n] = struct{}{}
	}

	lines := strings.Split(content, "\n")
	var out []string
	inTarget := false
	brackets := 0
	changed := false

	for _, line := range lines {
		t := strings.TrimSpace(line)
		if !inTarget {
			if strings.HasSuffix(t, "{") && t != "{" && !strings.HasPrefix(t, "#") {
				header := strings.TrimSpace(strings.TrimSuffix(t, "{"))
				matched := false
				for _, p := range strings.Split(header, ",") {
					if _, ok := targets[normalizeCaddyHost(strings.TrimSpace(p))]; ok {
						matched = true
						break
					}
				}
				if matched {
					inTarget = true
					brackets = strings.Count(line, "{") - strings.Count(line, "}")
					if brackets == 0 {
						inTarget = false
					}
					changed = true
					continue
				}
			}
			out = append(out, line)
			continue
		}

		brackets += strings.Count(line, "{")
		brackets -= strings.Count(line, "}")
		if brackets <= 0 {
			inTarget = false
			brackets = 0
		}
		changed = true
	}

	return strings.TrimSpace(strings.Join(out, "\n")) + "\n", changed
}

func splitFirstBlock(content, domain string) (string, string) {
	target := normalizeCaddyHost(domain)
	if target == "" {
		return "", content
	}
	lines := strings.Split(content, "\n")
	inBlock := false
	brackets := 0
	var block []string
	var rest []string

	for _, line := range lines {
		t := strings.TrimSpace(line)
		if !inBlock {
			if strings.HasSuffix(t, "{") && t != "{" && !strings.HasPrefix(t, "#") {
				header := strings.TrimSpace(strings.TrimSuffix(t, "{"))
				matched := false
				for _, p := range strings.Split(header, ",") {
					if normalizeCaddyHost(strings.TrimSpace(p)) == target {
						matched = true
						break
					}
				}
				if matched {
					inBlock = true
					brackets = strings.Count(line, "{") - strings.Count(line, "}")
					block = append(block, line)
					if brackets == 0 {
						inBlock = false
					}
					continue
				}
			}
			rest = append(rest, line)
			continue
		}

		block = append(block, line)
		brackets += strings.Count(line, "{")
		brackets -= strings.Count(line, "}")
		if brackets <= 0 {
			inBlock = false
			brackets = 0
			continue
		}
	}

	return strings.Join(block, "\n"), strings.Join(rest, "\n")
}

func appendResolveBlock(content, domain, proxy, otherDomains string) string {
	domain = strings.TrimSpace(domain)
	proxy = strings.TrimSpace(proxy)
	block := fmt.Sprintf("\n%s {\n\treverse_proxy /* %s\n}\n", domain, proxy)
	next := strings.TrimRight(content, "\n") + block

	target := buildCaddyRedirectTarget(domain)
	otherDomains = strings.ReplaceAll(otherDomains, ",", "\n")
	for _, ln := range strings.Split(otherDomains, "\n") {
		d := strings.TrimSpace(ln)
		if d == "" {
			continue
		}
		if caddyDomainExists(next, d) {
			continue
		}
		next += fmt.Sprintf("\n%s {\n\tredir %s{uri} permanent\n}\n", d, target)
	}
	return next
}

func buildCaddyRedirectTarget(domain string) string {
	target := strings.TrimSpace(domain)
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		return target
	}
	return "http://" + target
}

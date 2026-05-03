package api

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/utils/docker"
	"github.com/gofiber/fiber/v3"
	"strings"
	"sync"
	"time"
)

func AppInstallLogsStream(c fiber.Ctx) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")
	name := c.Params("name")
	if name == "" {
		return c.SendString("event: error\ndata: invalid app install name\n\n")
	}
	active := service.IsAppInstallLoggerActive(name)
	logger := service.GetAppInstallLogger(name)
	ch := logger.Subscribe()
	c.Status(200)
	ctxRaw := c.RequestCtx()
	ctxRaw.SetBodyStreamWriter(func(w *bufio.Writer) {
		defer logger.Unsubscribe(ch)
		for _, logMsg := range logger.GetLogs() {
			fmt.Fprintf(w, "data: %s\n\n", logMsg)
			if err := w.Flush(); err != nil {
				return
			}
		}
		if !active {
			fmt.Fprintf(w, "data: EOF\n\n")
			_ = w.Flush()
			return
		}
		for {
			select {
			case logMsg, ok := <-ch:
				trimmed := strings.TrimSpace(logMsg)
				if !ok || trimmed == "EOF" || trimmed == "[\"EOF\"]" || strings.HasSuffix(trimmed, " INFO: EOF") {
					fmt.Fprintf(w, "data: EOF\n\n")
					_ = w.Flush()
					return
				}
				fmt.Fprintf(w, "data: %s\n\n", logMsg)
				if err := w.Flush(); err != nil {
					return
				}
			case <-time.After(1 * time.Second):
				fmt.Fprintf(w, "event: ping\ndata: ping\n\n")
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	})
	return nil
}
func AppInstalledRuntimeLogsStream(c fiber.Ctx) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")
	name := strings.TrimSpace(c.Params("name"))
	if name == "" {
		return c.SendString("event: error\ndata: invalid app install name\n\n")
	}
	appInstall, err := getInstalledByName(name)
	if err != nil {
		return c.SendString("event: error\ndata: app install not found\n\n")
	}
	containerNames := splitAppInstallContainerNames(appInstall.ContainerName)
	if len(containerNames) == 0 {
		return c.SendString("event: error\ndata: container name is empty\n\n")
	}
	c.Status(200)
	ctxRaw := c.RequestCtx()
	ctxRaw.SetBodyStreamWriter(func(w *bufio.Writer) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan struct{})
		go func() {
			select {
			case <-ctxRaw.Done():
				cancel()
			case <-done:
			}
		}()
		defer close(done)
		writeLine := func(line string) {
			fmt.Fprintf(w, "data: %s\n\n", line)
			_ = w.Flush()
		}
		streamErr := streamInstalledContainerLogs(ctx, containerNames, writeLine)
		if streamErr != nil && ctx.Err() == nil {
			fmt.Fprintf(w, "data: [ERROR] %s\n\n", streamErr.Error())
			_ = w.Flush()
		}
		fmt.Fprintf(w, "data: EOF\n\n")
		_ = w.Flush()
	})
	return nil
}
func splitAppInstallContainerNames(raw string) []string {
	var names []string
	for _, item := range strings.Split(raw, ",") {
		name := strings.TrimSpace(strings.TrimPrefix(item, "/"))
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	return names
}
func streamInstalledContainerLogs(ctx context.Context, containerNames []string, onLine func(string)) error {
	if len(containerNames) == 0 {
		return errors.New("container name is empty")
	}
	if len(containerNames) == 1 {
		return streamSingleContainerLogs(ctx, containerNames[0], false, onLine)
	}
	lineCh := make(chan string, 64)
	errCh := make(chan error, len(containerNames))
	var wg sync.WaitGroup
	for _, name := range containerNames {
		containerName := name
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := streamSingleContainerLogs(ctx, containerName, true, func(line string) {
				select {
				case <-ctx.Done():
				case lineCh <- line:
				}
			}); err != nil && ctx.Err() == nil {
				errCh <- err
			}
		}()
	}
	go func() {
		wg.Wait()
		close(lineCh)
		close(errCh)
	}()
	var firstErr error
	for lineCh != nil || errCh != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case line, ok := <-lineCh:
			if !ok {
				lineCh = nil
				continue
			}
			onLine(line)
		case err, ok := <-errCh:
			if !ok {
				errCh = nil
				continue
			}
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}
func streamSingleContainerLogs(ctx context.Context, containerName string, prefix bool, onLine func(string)) error {
	cmd, err := docker.RuntimeCommand(ctx, "logs", "--tail", "200", "-f", containerName)
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return err
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var output strings.Builder
	for scanner.Scan() {
		line := strings.ReplaceAll(scanner.Text(), "\r", "")
		if line == "" {
			continue
		}
		if output.Len() > 0 {
			output.WriteByte('\n')
		}
		output.WriteString(line)
		if prefix {
			onLine(fmt.Sprintf("[%s] %s", containerName, line))
		} else {
			onLine(line)
		}
	}
	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		_ = cmd.Wait()
		return err
	}
	waitErr := cmd.Wait()
	if ctx.Err() != nil {
		return nil
	}
	if waitErr != nil {
		msg := strings.TrimSpace(output.String())
		if msg == "" {
			return waitErr
		}
		return fmt.Errorf("%w: %s", waitErr, msg)
	}
	return nil
}

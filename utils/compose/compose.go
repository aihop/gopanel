package compose

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	dockerRuntime "github.com/aihop/gopanel/utils/docker"
)

func resolveComposeCommand() (string, []string, error) {
	return dockerRuntime.DefaultRuntimeAdapter().ResolveComposeCommand(context.Background())
}

func Command(ctx context.Context, args ...string) (*exec.Cmd, error) {
	return dockerRuntime.DefaultRuntimeAdapter().ComposeCommand(ctx, args...)
}

func ResolveCommand() (string, []string, error) {
	return resolveComposeCommand()
}

func Exec(ctx context.Context, args ...string) (string, error) {
	c, err := Command(ctx, args...)
	if err != nil {
		return "", err
	}
	out, runErr := c.CombinedOutput()
	if runErr != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return string(out), runErr
		}
		return string(out), fmt.Errorf("%w: %s", runErr, msg)
	}
	return string(out), nil
}

func ExecStream(ctx context.Context, onLine func(string), args ...string) (string, error) {
	c, err := Command(ctx, args...)
	if err != nil {
		return "", err
	}
	stdout, err := c.StdoutPipe()
	if err != nil {
		return "", err
	}
	c.Stderr = c.Stdout
	if err := c.Start(); err != nil {
		return "", err
	}

	var buf bytes.Buffer
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if onLine != nil {
			onLine(line)
		}
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	scErr := sc.Err()
	waitErr := c.Wait()
	out := buf.String()
	if scErr != nil {
		return out, scErr
	}
	if waitErr != nil {
		msg := strings.TrimSpace(out)
		if msg == "" {
			return out, waitErr
		}
		return out, fmt.Errorf("%w: %s", waitErr, msg)
	}
	return out, nil
}

func ExecStreamWithEnv(ctx context.Context, onLine func(string), extraEnv []string, args ...string) (string, error) {
	c, err := Command(ctx, args...)
	if err != nil {
		return "", err
	}
	if len(extraEnv) > 0 {
		if len(c.Env) > 0 {
			c.Env = append(c.Env, extraEnv...)
		} else {
			c.Env = append(os.Environ(), extraEnv...)
		}
	}
	stdout, err := c.StdoutPipe()
	if err != nil {
		return "", err
	}
	c.Stderr = c.Stdout
	if err := c.Start(); err != nil {
		return "", err
	}

	var buf bytes.Buffer
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if onLine != nil {
			onLine(line)
		}
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	scErr := sc.Err()
	waitErr := c.Wait()
	out := buf.String()
	if scErr != nil {
		return out, scErr
	}
	if waitErr != nil {
		msg := strings.TrimSpace(out)
		if msg == "" {
			return out, waitErr
		}
		return out, fmt.Errorf("%w: %s", waitErr, msg)
	}
	return out, nil
}

func Pull(filePath string) (string, error) {
	return Exec(context.Background(), "-f", filePath, "pull")
}

func Up(filePath string) (string, error) {
	return Exec(context.Background(), "-f", filePath, "up", "-d")
}

func Down(filePath string) (string, error) {
	return Exec(context.Background(), "-f", filePath, "down", "--remove-orphans")
}

func Start(filePath string) (string, error) {
	return Exec(context.Background(), "-f", filePath, "start")
}

func Stop(filePath string) (string, error) {
	return Exec(context.Background(), "-f", filePath, "stop")
}

func Restart(filePath string) (string, error) {
	return Exec(context.Background(), "-f", filePath, "restart")
}

func Operate(filePath, operation string) (string, error) {
	opArgs := strings.Fields(operation)
	args := append([]string{"-f", filePath}, opArgs...)
	return Exec(context.Background(), args...)
}

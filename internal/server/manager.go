package server

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

func EnsureRunning() (string, error) {
	check := exec.Command("pgrep", "-f", "[o]pencode ser")
	if err := check.Run(); err == nil {
		return "http://127.0.0.1:4096", nil
	}

	cmd := exec.Command("opencode", "serve")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start: %w", err)
	}

	addrCh := make(chan string)
	errCh := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(io.MultiReader(stdout, stderr))
		re := regexp.MustCompile(`listening on (http://\S+)`)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if matches := re.FindStringSubmatch(line); len(matches) >= 2 {
				addrCh <- matches[1]
				return
			}
		}
		errCh <- cmd.Wait()
	}()

	select {
	case addr := <-addrCh:
		return addr, nil
	case err := <-errCh:
		return "", fmt.Errorf("server exited: %w", err)
	case <-time.After(5 * time.Second):
		return "", fmt.Errorf("timeout waiting for server address")
	}
}

func KillServer() error {
	cmd := exec.Command("pkill", "-f", "opencode ser")
	return cmd.Run()
}

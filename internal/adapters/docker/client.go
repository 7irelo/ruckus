package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Adapter struct {
	binPath string
}

type ContainerInfo struct {
	ID      string
	Name    string
	Image   string
	State   string
	Running bool
	Labels  map[string]string
}

type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type StressOptions struct {
	Name        string
	Image       string
	CPUWorkers  int
	NetworkMode string
	HostLevel   bool
	Target      string
}

func NewLocalAdapter() (*Adapter, error) {
	if err := validateDockerHost(); err != nil {
		return nil, err
	}

	binPath, err := exec.LookPath("docker")
	if err != nil {
		return nil, errors.New("docker CLI not found in PATH")
	}

	return &Adapter{binPath: binPath}, nil
}

func (a *Adapter) Ping(ctx context.Context) error {
	_, _, err := a.runDocker(ctx, "version", "--format", "{{.Server.Version}}")
	if err != nil {
		return fmt.Errorf("docker daemon unavailable: %w", err)
	}
	return nil
}

func (a *Adapter) ListEligibleContainers(ctx context.Context, labelKey string, labelValue string) ([]ContainerInfo, error) {
	filter := fmt.Sprintf("label=%s=%s", labelKey, labelValue)
	stdout, _, err := a.runDocker(ctx, "ps", "-a", "--filter", filter, "--format", "{{.ID}}")
	if err != nil {
		return nil, err
	}

	ids := splitNonEmptyLines(stdout)
	containers := make([]ContainerInfo, 0, len(ids))
	for _, id := range ids {
		info, inspectErr := a.InspectContainer(ctx, id)
		if inspectErr != nil {
			return nil, inspectErr
		}
		containers = append(containers, info)
	}

	sort.Slice(containers, func(i int, j int) bool {
		return containers[i].Name < containers[j].Name
	})

	return containers, nil
}

func (a *Adapter) InspectContainer(ctx context.Context, target string) (ContainerInfo, error) {
	stdout, _, err := a.runDocker(ctx, "inspect", "--type", "container", target)
	if err != nil {
		return ContainerInfo{}, err
	}

	var raw []inspectContainer
	if unmarshalErr := json.Unmarshal([]byte(stdout), &raw); unmarshalErr != nil {
		return ContainerInfo{}, fmt.Errorf("parse docker inspect output: %w", unmarshalErr)
	}
	if len(raw) == 0 {
		return ContainerInfo{}, fmt.Errorf("target %q not found", target)
	}

	inspected := raw[0]
	return ContainerInfo{
		ID:      inspected.ID,
		Name:    strings.TrimPrefix(inspected.Name, "/"),
		Image:   inspected.Config.Image,
		State:   inspected.State.Status,
		Running: inspected.State.Running,
		Labels:  inspected.Config.Labels,
	}, nil
}

func (a *Adapter) RestartContainer(ctx context.Context, target string) error {
	_, _, err := a.runDocker(ctx, "restart", target)
	return err
}

func (a *Adapter) StartContainer(ctx context.Context, target string) error {
	_, _, err := a.runDocker(ctx, "start", target)
	return err
}

func (a *Adapter) StopAndRemoveContainer(ctx context.Context, target string) error {
	_, stderr, err := a.runDocker(ctx, "rm", "-f", target)
	if err != nil {
		if strings.Contains(stderr, "No such container") {
			return nil
		}
		return err
	}
	return nil
}

func (a *Adapter) Exec(ctx context.Context, target string, command []string) (CommandResult, error) {
	args := make([]string, 0, 2+len(command))
	args = append(args, "exec", target)
	args = append(args, command...)

	cmd := exec.CommandContext(ctx, a.binPath, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := CommandResult{
		Stdout: strings.TrimSpace(stdout.String()),
		Stderr: strings.TrimSpace(stderr.String()),
	}
	if err == nil {
		result.ExitCode = 0
		return result, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
		return result, nil
	}

	return result, fmt.Errorf("run docker exec: %w", err)
}

func (a *Adapter) IsContainerRunning(ctx context.Context, target string) (bool, error) {
	info, err := a.InspectContainer(ctx, target)
	if err != nil {
		return false, err
	}
	return info.Running, nil
}

func (a *Adapter) IsTCNetemAvailable(ctx context.Context, target string) (bool, error) {
	result, err := a.Exec(ctx, target, []string{"tc", "qdisc", "show"})
	if err != nil {
		return false, err
	}
	if result.ExitCode == 0 {
		return true, nil
	}

	combined := strings.ToLower(result.Stdout + " " + result.Stderr)
	if strings.Contains(combined, "executable file not found") || strings.Contains(combined, "not found") {
		return false, nil
	}

	return true, nil
}

func (a *Adapter) ApplyNetem(ctx context.Context, target string, iface string, latency string, jitter string) error {
	if iface == "" {
		return errors.New("network interface is required")
	}

	command := []string{
		"tc", "qdisc", "replace", "dev", iface, "root", "netem", "delay", latency, jitter,
	}
	result, err := a.Exec(ctx, target, command)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("tc netem apply failed: %s", firstNonEmpty(result.Stderr, result.Stdout))
	}
	return nil
}

func (a *Adapter) ClearNetem(ctx context.Context, target string, iface string) error {
	if iface == "" {
		return errors.New("network interface is required")
	}

	command := []string{"tc", "qdisc", "del", "dev", iface, "root"}
	result, err := a.Exec(ctx, target, command)
	if err != nil {
		return err
	}
	if result.ExitCode == 0 {
		return nil
	}

	combined := strings.ToLower(result.Stdout + " " + result.Stderr)
	if strings.Contains(combined, "no such file") || strings.Contains(combined, "cannot find") {
		return nil
	}

	return fmt.Errorf("tc netem cleanup failed: %s", firstNonEmpty(result.Stderr, result.Stdout))
}

func (a *Adapter) RunStressContainer(ctx context.Context, options StressOptions) (string, error) {
	if options.Image == "" {
		options.Image = "progrium/stress"
	}
	if options.CPUWorkers <= 0 {
		options.CPUWorkers = 1
	}
	if options.Name == "" {
		options.Name = fmt.Sprintf("ruckus-stress-%d", time.Now().UnixNano())
	}

	args := []string{"run", "-d", "--name", options.Name}
	if options.HostLevel {
		args = append(args, "--network", "host", "--pid", "host", "--privileged")
	} else if options.NetworkMode != "" {
		args = append(args, "--network", options.NetworkMode)
	}
	args = append(args, options.Image, "--cpu", strconv.Itoa(options.CPUWorkers))

	stdout, _, err := a.runDocker(ctx, args...)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(stdout), nil
}

func (a *Adapter) runDocker(ctx context.Context, args ...string) (string, string, error) {
	cmd := exec.CommandContext(ctx, a.binPath, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			message := strings.TrimSpace(stderr.String())
			if message == "" {
				message = strings.TrimSpace(stdout.String())
			}
			return "", strings.TrimSpace(stderr.String()), fmt.Errorf("docker %s failed (exit %d): %s", strings.Join(args, " "), exitErr.ExitCode(), message)
		}
		return "", strings.TrimSpace(stderr.String()), fmt.Errorf("run docker %s: %w", strings.Join(args, " "), err)
	}

	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), nil
}

func validateDockerHost() error {
	host := strings.TrimSpace(os.Getenv("DOCKER_HOST"))
	if host == "" {
		return nil
	}

	if strings.HasPrefix(host, "unix://") || strings.HasPrefix(host, "npipe://") {
		return nil
	}

	return fmt.Errorf("only local Docker engines are supported in v1, refusing DOCKER_HOST=%q", host)
}

func splitNonEmptyLines(text string) []string {
	rawLines := strings.Split(text, "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		lines = append(lines, trimmed)
	}
	return lines
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

type inspectContainer struct {
	ID     string `json:"Id"`
	Name   string `json:"Name"`
	Config struct {
		Image  string            `json:"Image"`
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`
	State struct {
		Status  string `json:"Status"`
		Running bool   `json:"Running"`
	} `json:"State"`
}

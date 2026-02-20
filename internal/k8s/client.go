package k8s

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type Client struct {
	kubeconfig string
}

type DiscoveredServer struct {
	Name      string
	Game      string
	Namespace string
	Replicas  int
}

type DeploymentResources struct {
	MemoryRequest string
	MemoryLimit   string
	CPURequest    string
	CPULimit      string
}

type PVCInfo struct {
	StorageRequest string
	StorageClass   string
}

func NewClient(kubeconfig string) *Client {
	return &Client{
		kubeconfig: kubeconfig,
	}
}

func (c *Client) kubectl(args ...string) *exec.Cmd {
	cmd := exec.Command("kubectl", args...)
	cmd.Env = append(os.Environ(), "KUBECONFIG="+c.kubeconfig)
	return cmd
}

func (c *Client) zcloudK(args ...string) *exec.Cmd {
	zcloudArgs := append([]string{"k"}, args...)
	cmd := exec.Command("zcloud", zcloudArgs...)
	cmd.Env = append(os.Environ(), "KUBECONFIG="+c.kubeconfig)
	return cmd
}

func (c *Client) preferZcloudExec() bool {
	if !strings.Contains(c.kubeconfig, ".zcloud") {
		return false
	}
	_, err := exec.LookPath("zcloud")
	return err == nil
}

func (c *Client) execTransport(args ...string) *exec.Cmd {
	if c.preferZcloudExec() {
		return c.zcloudK(args...)
	}
	return c.kubectl(args...)
}

func (c *Client) Apply(manifest string) error {
	cmd := c.kubectl("apply", "-f", "-")
	cmd.Stdin = strings.NewReader(manifest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (c *Client) ApplyAll(manifests []string) error {
	combined := strings.Join(manifests, "\n---\n")
	return c.Apply(combined)
}

func (c *Client) DeleteNamespace(namespace string) error {
	cmd := c.kubectl("delete", "namespace", namespace, "--ignore-not-found")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (c *Client) NamespaceExists(namespace string) bool {
	cmd := c.kubectl("get", "namespace", namespace)
	return cmd.Run() == nil
}

func (c *Client) GetPodStatus(namespace, labelSelector string) (string, error) {
	args := []string{"get", "pods", "-n", namespace}
	if strings.TrimSpace(labelSelector) != "" {
		args = append(args, "-l", labelSelector)
	}
	args = append(args, "-o", "jsonpath={.items[0].status.phase}")

	cmd := c.kubectl(args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (c *Client) GetPodName(namespace, labelSelector string) (string, error) {
	cmd := c.kubectl("get", "pods", "-n", namespace,
		"-l", labelSelector,
		"-o", "jsonpath={.items[0].metadata.name}")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (c *Client) WaitForReady(namespace, deployment string, timeoutSeconds int) error {
	cmd := c.kubectl("wait", "--for=condition=available",
		fmt.Sprintf("deployment/%s", deployment),
		"-n", namespace,
		fmt.Sprintf("--timeout=%ds", timeoutSeconds))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (c *Client) RunJob(namespace, jobName, jobManifest string, timeoutSeconds int) error {
	if err := validateJobArgs(namespace, jobName, timeoutSeconds); err != nil {
		return err
	}

	if err := c.Apply(jobManifest); err != nil {
		return fmt.Errorf("applying job: %w", err)
	}

	return c.waitForJobCompletion(namespace, jobName, timeoutSeconds)
}

func (c *Client) RunBackupJob(namespace, jobName, jobManifest string, timeoutSeconds int) error {
	if err := c.RunJob(namespace, jobName, jobManifest, timeoutSeconds); err != nil {
		return fmt.Errorf("running backup job: %w", err)
	}
	return nil
}

func (c *Client) RunJobAndGetLogs(namespace, jobName, jobManifest string, timeoutSeconds int) (string, error) {
	if err := c.RunJob(namespace, jobName, jobManifest, timeoutSeconds); err != nil {
		return "", err
	}

	logsCmd := c.kubectl("logs", fmt.Sprintf("job/%s", jobName), "-n", namespace)
	out, err := logsCmd.Output()
	if err != nil {
		return "", fmt.Errorf("reading job logs: %w", err)
	}

	return string(out), nil
}

func validateJobArgs(namespace, jobName string, timeoutSeconds int) error {
	if strings.TrimSpace(namespace) == "" {
		return fmt.Errorf("namespace is required")
	}
	if strings.TrimSpace(jobName) == "" {
		return fmt.Errorf("job name is required")
	}
	if timeoutSeconds <= 0 {
		return fmt.Errorf("timeout must be greater than 0")
	}
	return nil
}

func (c *Client) waitForJobCompletion(namespace, jobName string, timeoutSeconds int) error {
	waitCmd := c.kubectl("wait", "--for=condition=complete",
		fmt.Sprintf("job/%s", jobName),
		"-n", namespace,
		fmt.Sprintf("--timeout=%ds", timeoutSeconds))
	waitCmd.Stdout = os.Stdout
	waitCmd.Stderr = os.Stderr

	if err := waitCmd.Run(); err != nil {
		failed, message, statusErr := c.jobFailed(namespace, jobName)
		if statusErr == nil && failed {
			if message != "" {
				return fmt.Errorf("job failed: %s", message)
			}
			return fmt.Errorf("job failed")
		}
		return fmt.Errorf("waiting for job completion: %w", err)
	}

	return nil
}

func (c *Client) ScaleDeployment(namespace, deployment string, replicas int) error {
	cmd := c.kubectl("scale", "deployment", deployment,
		"-n", namespace,
		fmt.Sprintf("--replicas=%d", replicas))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (c *Client) GetReplicas(namespace, deployment string) (int, error) {
	cmd := c.kubectl("get", "deployment", deployment,
		"-n", namespace,
		"-o", "jsonpath={.spec.replicas}")
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	replicas, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0, err
	}
	return replicas, nil
}

func (c *Client) GetPodNodeName(namespace, labelSelector string) (string, error) {
	return c.getPodField(namespace, labelSelector, "jsonpath={.items[0].spec.nodeName}")
}

func (c *Client) GetPodStartTime(namespace, labelSelector string) (string, error) {
	return c.getPodField(namespace, labelSelector, "jsonpath={.items[0].status.startTime}")
}

func (c *Client) GetPodTop(namespace, labelSelector string) (cpu string, memory string, err error) {
	args := []string{"top", "pod", "-n", namespace}
	if strings.TrimSpace(labelSelector) != "" {
		args = append(args, "-l", labelSelector)
	}
	args = append(args, "--no-headers")

	cmd := c.kubectl(args...)
	out, err := cmd.Output()
	if err != nil {
		return "", "", err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "No resources found") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		return fields[1], fields[2], nil
	}

	return "", "", fmt.Errorf("no pod metrics available")
}

func (c *Client) GetDeploymentResources(namespace, deployment string) (DeploymentResources, error) {
	cmd := c.kubectl("get", "deployment", deployment,
		"-n", namespace,
		"-o", "jsonpath={.spec.template.spec.containers[0].resources.requests.memory},{.spec.template.spec.containers[0].resources.limits.memory},{.spec.template.spec.containers[0].resources.requests.cpu},{.spec.template.spec.containers[0].resources.limits.cpu}")
	out, err := cmd.Output()
	if err != nil {
		return DeploymentResources{}, err
	}

	result := strings.TrimSpace(string(out))
	parts := strings.SplitN(result, ",", 4)
	if len(parts) != 4 {
		return DeploymentResources{}, fmt.Errorf("invalid deployment resources output: %q", result)
	}

	return DeploymentResources{
		MemoryRequest: cleanJSONPathValue(parts[0]),
		MemoryLimit:   cleanJSONPathValue(parts[1]),
		CPURequest:    cleanJSONPathValue(parts[2]),
		CPULimit:      cleanJSONPathValue(parts[3]),
	}, nil
}

func (c *Client) GetPVCInfo(namespace string) (PVCInfo, error) {
	cmd := c.kubectl("get", "pvc", "-n", namespace,
		"-o", "jsonpath={.items[0].spec.resources.requests.storage},{.items[0].spec.storageClassName}")
	out, err := cmd.Output()
	if err != nil {
		return PVCInfo{}, err
	}

	result := strings.TrimSpace(string(out))
	parts := strings.SplitN(result, ",", 2)
	if len(parts) != 2 {
		return PVCInfo{}, fmt.Errorf("invalid pvc info output: %q", result)
	}

	return PVCInfo{
		StorageRequest: cleanJSONPathValue(parts[0]),
		StorageClass:   cleanJSONPathValue(parts[1]),
	}, nil
}

func (c *Client) HasBackupCronJob(namespace string) (bool, error) {
	cmd := c.kubectl("get", "cronjob", "-n", namespace,
		"-o", "jsonpath={.items[*].metadata.name}")
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}

	cronJobNames := strings.Fields(cleanJSONPathValue(string(out)))
	return len(cronJobNames) > 0, nil
}

func (c *Client) GetLastBackupTimestamp(namespace string) (string, error) {
	cmd := c.kubectl("get", "jobs", "-n", namespace,
		"-l", "type=backup",
		"--sort-by=.metadata.creationTimestamp",
		"-o", "jsonpath={.items[-1:].metadata.creationTimestamp}")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return cleanJSONPathValue(string(out)), nil
}

func (c *Client) getPodField(namespace, labelSelector, output string) (string, error) {
	args := []string{"get", "pods", "-n", namespace}
	if strings.TrimSpace(labelSelector) != "" {
		args = append(args, "-l", labelSelector)
	}
	args = append(args, "-o", output)

	cmd := c.kubectl(args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return cleanJSONPathValue(string(out)), nil
}

func cleanJSONPathValue(value string) string {
	cleaned := strings.TrimSpace(value)
	if cleaned == "" || cleaned == "<no value>" {
		return ""
	}
	return cleaned
}

func (c *Client) AttachConsole(namespace, deployment string) error {
	cmd := c.execTransport("attach", "-it",
		fmt.Sprintf("deployment/%s", deployment),
		"-n", namespace)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (c *Client) Logs(namespace, deployment string, follow bool) error {
	args := []string{"logs", fmt.Sprintf("deployment/%s", deployment), "-n", namespace}
	if follow {
		args = append(args, "-f")
	}
	cmd := c.kubectl(args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (c *Client) Exec(namespace, deployment string, command []string) error {
	args := []string{"exec", "-it",
		fmt.Sprintf("deployment/%s", deployment),
		"-n", namespace, "--"}
	args = append(args, command...)

	cmd := c.execTransport(args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (c *Client) ExecNoTTY(namespace, deployment string, command []string) error {
	args := []string{"exec",
		fmt.Sprintf("deployment/%s", deployment),
		"-n", namespace, "--"}
	args = append(args, command...)

	cmd := c.execTransport(args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (c *Client) GetDeployments(labelSelector string) ([]string, error) {
	cmd := c.kubectl("get", "deployments", "--all-namespaces",
		"-l", labelSelector,
		"-o", "jsonpath={.items[*].metadata.namespace}")

	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	result := strings.TrimSpace(out.String())
	if result == "" {
		return []string{}, nil
	}

	return strings.Split(result, " "), nil
}

func (c *Client) DiscoverServers() ([]DiscoveredServer, error) {
	cmd := c.kubectl("get", "deployments", "--all-namespaces",
		"-l", "app=zplay",
		"-o", `jsonpath={range .items[*]}{.metadata.namespace},{.metadata.labels.server},{.metadata.labels.game},{.spec.replicas}{"\n"}{end}`)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	result := strings.TrimSpace(string(out))
	if result == "" {
		return []DiscoveredServer{}, nil
	}

	lines := strings.Split(result, "\n")
	discovered := make([]DiscoveredServer, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ",", 4)
		if len(parts) != 4 {
			return nil, fmt.Errorf("invalid discovery output line: %q", line)
		}

		namespace := strings.TrimSpace(parts[0])
		name := strings.TrimSpace(parts[1])
		game := strings.TrimSpace(parts[2])
		replicasRaw := strings.TrimSpace(parts[3])

		replicas := 1
		if replicasRaw != "" && replicasRaw != "<no value>" {
			parsed, parseErr := strconv.Atoi(replicasRaw)
			if parseErr != nil {
				return nil, fmt.Errorf("parsing replicas in discovery output %q: %w", line, parseErr)
			}
			replicas = parsed
		}

		if namespace == "" || name == "" || game == "" {
			continue
		}

		discovered = append(discovered, DiscoveredServer{
			Name:      name,
			Game:      game,
			Namespace: namespace,
			Replicas:  replicas,
		})
	}

	return discovered, nil
}

func (c *Client) IsConnected() bool {
	cmd := c.kubectl("cluster-info")
	return cmd.Run() == nil
}

func (c *Client) jobFailed(namespace, jobName string) (bool, string, error) {
	cmd := c.kubectl("get", "job", jobName, "-n", namespace,
		"-o", `jsonpath={.status.conditions[?(@.type=="Failed")].status}:{.status.conditions[?(@.type=="Failed")].message}`)
	out, err := cmd.Output()
	if err != nil {
		return false, "", err
	}

	result := strings.TrimSpace(string(out))
	if result == "" {
		return false, "", nil
	}

	parts := strings.SplitN(result, ":", 2)
	if parts[0] != "True" {
		return false, "", nil
	}

	if len(parts) == 2 {
		return true, strings.TrimSpace(parts[1]), nil
	}

	return true, "", nil
}

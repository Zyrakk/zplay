package k8s

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Zyrakk/zplay/internal/config"
)

type Client struct {
	kubeconfig string
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		kubeconfig: cfg.Kubeconfig,
	}
}

func (c *Client) kubectl(args ...string) *exec.Cmd {
	cmd := exec.Command("kubectl", args...)
	cmd.Env = append(os.Environ(), "KUBECONFIG="+c.kubeconfig)
	return cmd
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

func (c *Client) GetPodStatus(namespace string) (string, error) {
	cmd := c.kubectl("get", "pods", "-n", namespace,
		"-o", "jsonpath={.items[0].status.phase}")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
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

func (c *Client) AttachConsole(namespace, deployment string) error {
	cmd := c.kubectl("attach", "-it",
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

	cmd := c.kubectl(args...)
	cmd.Stdin = os.Stdin
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

func (c *Client) IsConnected() bool {
	cmd := c.kubectl("cluster-info")
	return cmd.Run() == nil
}

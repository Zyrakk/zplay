package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

type ServerStatus struct {
	Name       string  `json:"name"`
	Game       string  `json:"game"`
	Variant    string  `json:"variant"`
	Status     string  `json:"status"`
	Node       string  `json:"node"`
	Port       int     `json:"port"`
	Players    Players `json:"players"`
	Uptime     string  `json:"uptime"`
	Memory     Memory  `json:"memory"`
	CPU        CPU     `json:"cpu"`
	LastBackup string  `json:"last_backup"`
	AutoBackup bool    `json:"auto_backup"`
}

type Players struct {
	Current int `json:"current"`
	Max     int `json:"max"`
}

type Memory struct {
	Used    string `json:"used"`
	Request string `json:"request"`
	Limit   string `json:"limit"`
}

type CPU struct {
	Used  string `json:"used"`
	Limit string `json:"limit"`
}

type Collector struct {
	clientset     *kubernetes.Clientset
	metricsClient *metricsv.Clientset
	cache         []ServerStatus
	cacheMu       sync.RWMutex
	cacheTime     time.Time
	cacheTTL      time.Duration
}

func NewCollector(clientset *kubernetes.Clientset, metricsClient *metricsv.Clientset) *Collector {
	return &Collector{
		clientset:     clientset,
		metricsClient: metricsClient,
		cacheTTL:      15 * time.Second,
	}
}

func (c *Collector) GetServers() []ServerStatus {
	c.cacheMu.RLock()
	if time.Since(c.cacheTime) < c.cacheTTL && c.cache != nil {
		defer c.cacheMu.RUnlock()
		return c.cache
	}
	c.cacheMu.RUnlock()

	servers := c.collect()

	c.cacheMu.Lock()
	c.cache = servers
	c.cacheTime = time.Now()
	c.cacheMu.Unlock()

	return servers
}

func (c *Collector) collect() []ServerStatus {
	ctx := context.Background()

	deployments, err := c.clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{
		LabelSelector: "app=zplay",
	})
	if err != nil {
		return nil
	}

	var servers []ServerStatus

	for _, dep := range deployments.Items {
		labels := dep.Labels
		name := labels["server"]
		game := labels["game"]
		if name == "" || game == "" {
			continue
		}

		namespace := dep.Namespace
		replicas := int32(1)
		if dep.Spec.Replicas != nil {
			replicas = *dep.Spec.Replicas
		}

		status := "Unknown"
		if replicas == 0 {
			status = "Stopped"
		}

		variant := ""
		node := ""
		port := 0
		maxPlayers := 0
		memRequest := ""
		memLimit := ""
		cpuLimit := ""

		if len(dep.Spec.Template.Spec.Containers) > 0 {
			container := dep.Spec.Template.Spec.Containers[0]

			image := container.Image
			if strings.Contains(image, "tmodloader") {
				variant = "tmodloader"
			} else if strings.Contains(image, "terraria") {
				variant = "vanilla"
			}

			if req, ok := container.Resources.Requests["memory"]; ok {
				memRequest = req.String()
			}
			if lim, ok := container.Resources.Limits["memory"]; ok {
				memLimit = lim.String()
			}
			if lim, ok := container.Resources.Limits["cpu"]; ok {
				cpuLimit = lim.String()
			}

			for _, env := range container.Env {
				switch env.Name {
				case "maxplayers", "TMOD_MAXPLAYERS":
					fmt.Sscanf(env.Value, "%d", &maxPlayers)
				}
			}
		}

		if ns := dep.Spec.Template.Spec.NodeSelector; ns != nil {
			node = ns["kubernetes.io/hostname"]
		}

		pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("app=zplay,server=%s,!job-name", name),
		})
		if err == nil && len(pods.Items) > 0 {
			pod := pods.Items[0]
			if replicas > 0 {
				status = string(pod.Status.Phase)
			}
			if pod.Spec.NodeName != "" {
				node = pod.Spec.NodeName
			}
		}

		uptime := ""
		if err == nil && len(pods.Items) > 0 {
			pod := pods.Items[0]
			if pod.Status.StartTime != nil {
				uptime = formatUptime(pod.Status.StartTime.Time)
			}
		}

		services, err := c.clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("app=zplay,server=%s", name),
		})
		if err == nil && len(services.Items) > 0 {
			svc := services.Items[0]
			if len(svc.Spec.Ports) > 0 {
				port = int(svc.Spec.Ports[0].Port)
			}
		}

		memUsed := ""
		cpuUsed := ""
		if c.metricsClient != nil {
			podMetrics, err := c.metricsClient.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{
				LabelSelector: fmt.Sprintf("app=zplay,server=%s,!job-name", name),
			})
			if err == nil && len(podMetrics.Items) > 0 {
				for _, container := range podMetrics.Items[0].Containers {
					memUsed = formatMemoryGi(container.Usage.Memory().Value())
					cpuUsed = formatCPUCores(container.Usage.Cpu().MilliValue())
				}
			}
		}

		autoBackup := false
		lastBackup := ""
		cronjobs, err := c.clientset.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
		if err == nil && len(cronjobs.Items) > 0 {
			autoBackup = true
		}
		jobs, err := c.clientset.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
		if err == nil {
			for _, job := range jobs.Items {
				if strings.Contains(job.Name, "-backup-") && !strings.Contains(job.Name, "-backup-list-") {
					ts := job.CreationTimestamp.Format(time.RFC3339)
					if ts > lastBackup {
						lastBackup = ts
					}
				}
			}
		}

		servers = append(servers, ServerStatus{
			Name:       name,
			Game:       game,
			Variant:    variant,
			Status:     status,
			Node:       node,
			Port:       port,
			Players:    Players{Current: 0, Max: maxPlayers},
			Uptime:     uptime,
			Memory:     Memory{Used: memUsed, Request: memRequest, Limit: memLimit},
			CPU:        CPU{Used: cpuUsed, Limit: cpuLimit},
			LastBackup: lastBackup,
			AutoBackup: autoBackup,
		})
	}

	return servers
}

func formatUptime(start time.Time) string {
	elapsed := time.Since(start)
	if elapsed < 0 {
		return "N/A"
	}

	days := int(elapsed.Hours()) / 24
	hours := int(elapsed.Hours()) % 24
	minutes := int(elapsed.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

func formatMemoryGi(bytes int64) string {
	gi := float64(bytes) / (1024 * 1024 * 1024)
	if gi >= 1 {
		return fmt.Sprintf("%.1fGi", gi)
	}
	mi := float64(bytes) / (1024 * 1024)
	return fmt.Sprintf("%.0fMi", mi)
}

func formatCPUCores(milliCores int64) string {
	cores := float64(milliCores) / 1000
	return fmt.Sprintf("%.1f", cores)
}

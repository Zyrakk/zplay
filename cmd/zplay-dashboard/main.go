package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

//go:embed static
var staticFiles embed.FS

var Version = "dev"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	config, err := buildConfig()
	if err != nil {
		log.Fatalf("Failed to build Kubernetes config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	metricsClient, err := metricsv.NewForConfig(config)
	if err != nil {
		log.Printf("Warning: metrics client unavailable: %v", err)
		metricsClient = nil
	}

	collector := NewCollector(clientset, metricsClient)

	mux := http.NewServeMux()

	mux.HandleFunc("/api/servers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		servers := collector.GetServers()
		if servers == nil {
			servers = []ServerStatus{}
		}
		json.NewEncoder(w).Encode(servers)
	})

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("Failed to create static FS: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log.Printf("zplay-dashboard %s starting on :%s", Version, port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func buildConfig() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, _ := os.UserHomeDir()
		candidates := []string{
			home + "/.zcloud/kubeconfig",
			home + "/.kube/config",
		}
		for _, path := range candidates {
			if _, err := os.Stat(path); err == nil {
				kubeconfig = path
				break
			}
		}
	}

	if kubeconfig == "" {
		return nil, fmt.Errorf("no kubeconfig found and not running in-cluster")
	}

	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}

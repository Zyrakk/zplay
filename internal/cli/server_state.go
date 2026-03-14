package cli

import (
	"fmt"
	"strings"

	"github.com/Zyrakk/zplay/internal/config"
	"github.com/Zyrakk/zplay/internal/games"
	"github.com/Zyrakk/zplay/internal/k8s"
)

func resolveNamespace(srv config.ServerInfo, game games.Game) string {
	if srv.Namespace != "" {
		return srv.Namespace
	}
	return game.GetNamespace(srv.Name)
}

func resolveServerStatus(srv config.ServerInfo, client *k8s.Client) string {
	game := games.Get(srv.Game)
	if game == nil {
		return "Unknown"
	}

	namespace := resolveNamespace(srv, game)
	deployment := game.GetDeploymentName(srv.Name)
	status, _ := client.GetPodStatus(namespace, fmt.Sprintf("app=zplay,server=%s,!job-name", srv.Name))
	if status == "" {
		status = "Unknown"
	}

	replicas, err := client.GetReplicas(namespace, deployment)
	if err == nil && replicas == 0 {
		status = "Stopped"
	}

	return status
}

func collectServerRows(state *config.ServerState, client *k8s.Client) []listServerJSON {
	entries := make([]listServerJSON, 0, len(state.Servers))
	for _, srv := range state.Servers {
		node := srv.Node
		if node == "" {
			node = "auto"
		}

		entries = append(entries, listServerJSON{
			Name:    srv.Name,
			Game:    srv.Game,
			Variant: srv.Variant,
			Port:    srv.Port,
			Node:    node,
			Status:  resolveServerStatus(srv, client),
		})
	}
	return entries
}

func findServerByName(state *config.ServerState, name string) (config.ServerInfo, error) {
	target := strings.TrimSpace(name)
	if target == "" {
		return config.ServerInfo{}, fmt.Errorf("server name is required")
	}

	for _, srv := range state.Servers {
		if srv.Name == target {
			return srv, nil
		}
	}

	return config.ServerInfo{}, fmt.Errorf("server '%s' not found", target)
}

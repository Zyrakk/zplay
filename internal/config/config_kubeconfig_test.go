package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveKubeconfig_ExplicitCustomConfigWins(t *testing.T) {
	home := t.TempDir()
	explicit := filepath.Join(home, "custom", "cluster.yaml")

	got := resolveKubeconfig(explicit, home)
	if got != explicit {
		t.Fatalf("expected explicit kubeconfig %q, got %q", explicit, got)
	}
}

func TestResolveKubeconfig_LegacyDefaultFallsBackToEnv(t *testing.T) {
	home := t.TempDir()
	legacy := filepath.Join(home, ".zcloud", "kubeconfig")
	envPath := filepath.Join(home, "env", "kubeconfig")

	t.Setenv("KUBECONFIG", envPath)

	got := resolveKubeconfig(legacy, home)
	if got != envPath {
		t.Fatalf("expected env kubeconfig %q, got %q", envPath, got)
	}
}

func TestResolveKubeconfig_UsesHomeKubeconfigWhenAvailable(t *testing.T) {
	home := t.TempDir()
	legacy := filepath.Join(home, ".zcloud", "kubeconfig")
	kubeconfig := filepath.Join(home, ".kube", "config")

	t.Setenv("KUBECONFIG", "")
	if err := os.MkdirAll(filepath.Dir(kubeconfig), 0755); err != nil {
		t.Fatalf("creating kubeconfig dir: %v", err)
	}
	if err := os.WriteFile(kubeconfig, []byte("apiVersion: v1\n"), 0644); err != nil {
		t.Fatalf("writing kubeconfig: %v", err)
	}

	got := resolveKubeconfig(legacy, home)
	if got != kubeconfig {
		t.Fatalf("expected ~/.kube/config %q, got %q", kubeconfig, got)
	}
}

func TestResolveKubeconfig_UsesLegacyWhenItExists(t *testing.T) {
	home := t.TempDir()
	legacy := filepath.Join(home, ".zcloud", "kubeconfig")

	t.Setenv("KUBECONFIG", "")
	if err := os.MkdirAll(filepath.Dir(legacy), 0755); err != nil {
		t.Fatalf("creating legacy kubeconfig dir: %v", err)
	}
	if err := os.WriteFile(legacy, []byte("apiVersion: v1\n"), 0644); err != nil {
		t.Fatalf("writing legacy kubeconfig: %v", err)
	}

	got := resolveKubeconfig(legacy, home)
	if got != legacy {
		t.Fatalf("expected legacy kubeconfig %q, got %q", legacy, got)
	}
}

func TestResolveKubeconfig_FallsBackToLegacyPathWhenNothingElseExists(t *testing.T) {
	home := t.TempDir()
	legacy := filepath.Join(home, ".zcloud", "kubeconfig")

	t.Setenv("KUBECONFIG", "")

	got := resolveKubeconfig("", home)
	if got != legacy {
		t.Fatalf("expected legacy fallback path %q, got %q", legacy, got)
	}
}


package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Zyrakk/zplay/internal/config"
	"github.com/Zyrakk/zplay/internal/k8s"
)

func RunCleanup(cfg *config.Config) error {
	clearScreen()
	fmt.Println(titleStyle.Render("Cleanup Resources"))
	fmt.Println()

	client := k8s.NewClient(cfg.Kubeconfig)
	pvs, err := client.GetReleasedPVs()
	if err != nil {
		return fmt.Errorf("listing released PVs: %w", err)
	}

	if len(pvs) == 0 {
		fmt.Println(dimStyle.Render("No orphaned resources found."))
		return nil
	}

	fmt.Printf("Found %d orphaned PersistentVolume(s):\n\n", len(pvs))
	for _, pv := range pvs {
		fmt.Printf("  %s (%s, %s)\n", pv.Name, pv.Size, pv.Namespace)
	}
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Delete all orphaned PVs? [y/N]: ")
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))
	if confirm != "y" && confirm != "yes" {
		fmt.Println("Cancelled.")
		return nil
	}

	deleted := 0
	for _, pv := range pvs {
		printInfo(fmt.Sprintf("Deleting %s...", pv.Name))
		if err := client.DeletePV(pv.Name); err != nil {
			printError(fmt.Sprintf("Failed to delete %s: %s", pv.Name, err.Error()))
			continue
		}
		deleted++
	}

	printSuccess(fmt.Sprintf("Deleted %d/%d orphaned PVs", deleted, len(pvs)))
	return nil
}

type CleanupOptions struct {
	Yes    bool
	DryRun bool
	JSON   bool
}

func RunCleanupNonInteractive(cfg *config.Config, opts CleanupOptions) error {
	client := k8s.NewClient(cfg.Kubeconfig)
	pvs, err := client.GetReleasedPVs()
	if err != nil {
		return fmt.Errorf("listing released PVs: %w", err)
	}

	if len(pvs) == 0 {
		if opts.JSON {
			fmt.Println("[]")
		} else {
			fmt.Println("No orphaned resources found.")
		}
		return nil
	}

	if opts.JSON {
		type pvJSON struct {
			Name      string `json:"name"`
			Size      string `json:"size"`
			Namespace string `json:"namespace"`
		}
		items := make([]pvJSON, 0, len(pvs))
		for _, pv := range pvs {
			items = append(items, pvJSON{Name: pv.Name, Size: pv.Size, Namespace: pv.Namespace})
		}
		data, _ := json.MarshalIndent(items, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Found %d orphaned PersistentVolume(s):\n", len(pvs))
	for _, pv := range pvs {
		fmt.Printf("  %s (%s, %s)\n", pv.Name, pv.Size, pv.Namespace)
	}

	if opts.DryRun {
		fmt.Println("Dry run — no resources deleted.")
		return nil
	}

	if !opts.Yes {
		return fmt.Errorf("cleanup requires --yes to confirm deletion")
	}

	deleted := 0
	for _, pv := range pvs {
		fmt.Printf("Deleting %s...\n", pv.Name)
		if err := client.DeletePV(pv.Name); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to delete %s: %v\n", pv.Name, err)
			continue
		}
		deleted++
	}

	fmt.Printf("Deleted %d/%d orphaned PVs\n", deleted, len(pvs))
	return nil
}

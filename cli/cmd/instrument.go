/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/odigos-io/odigos/cli/pkg/lifecycle"

	"github.com/odigos-io/odigos/common/consts"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/odigos-io/odigos/cli/pkg/kube"

	"github.com/odigos-io/odigos/cli/pkg/preflight"

	"github.com/spf13/cobra"
)

const (
	excludeNamespacesFileFlag = "exclude-namespaces-file"
	excludeAppsFileFlag       = "exclude-apps-file"
	skipPreflightCheckFlag    = "skip-preflight-checks"
)

// instrumentCmd represents the instrument command
var instrumentCmd = &cobra.Command{
	Use:   "instrument",
	Short: "Instrument applications with Odigos",
	Long: `Instrument applications with Odigos. This command will instrument the application with Odigos CLI
and monitor the instrumentation status.`,
}

// clusterCmd represents the cluster command
var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Instrument entire cluster with Odigos",
	Long: `Instrument entire cluster with Odigos. This command will instrument the entire Kubernetes cluster with
Odigos CLI and monitor the instrumentation status.`,
	Run: func(cmd *cobra.Command, args []string) {
		excludedNs, err := readFileLines(cmd.Flag(excludeNamespacesFileFlag).Value.String())
		if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Cannot read exclude-namespaces-file: %s\n", err)
			os.Exit(1)
		}

		excludedApps, err := readFileLines(cmd.Flag(excludeAppsFileFlag).Value.String())
		if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Cannot read exclude-apps-file: %s\n", err)
			os.Exit(1)
		}

		fmt.Printf("About to instrument an entire cluster with Odigos\n")
		fmt.Printf("Excluded Namespaces:   %d\n", len(excludedNs))
		fmt.Printf("Excluded Applications: %d\n", len(excludedApps))
		fmt.Printf("%-50s", "Checking if Kubernetes cluster is reachable")
		client, err := kube.CreateClient(cmd)
		if err != nil {
			fmt.Printf("\u001B[31mERROR\u001B[0m\n\n")
			fmt.Printf("Check failed: %s\n", err)
			os.Exit(1)
		} else {
			fmt.Printf("\u001B[32mPASS\u001B[0m\n\n")
		}

		runPreflightChecks(cmd.Context(), cmd, client)

		fmt.Printf("Starting instrumentation ...\n")
		instrumentCluster(cmd.Context(), client, excludedNs, excludedApps)
	},
}

func instrumentCluster(ctx context.Context, client *kube.Client, excludedNs, excludedApps map[string]struct{}) {
	systemNs := sliceToMap(consts.SystemNamespaces)
	nsList, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Printf("\033[31mERROR\033[0m Cannot list namespaces: %s\n", err)
		os.Exit(1)
	}

	orchestrator, err := lifecycle.NewOrchestrator(client, ctx)
	if err != nil {
		fmt.Printf("\033[31mERROR\033[0m Cannot create orchestrator: %s\n", err)
		os.Exit(1)
	}

	for _, ns := range nsList.Items {
		fmt.Printf("Instrumenting namespace: %s\n", ns.Name)
		_, excluded := excludedNs[ns.Name]
		_, system := systemNs[ns.Name]
		if excluded || system {
			fmt.Printf("  - Skipping namespace due to exclusion file or system namespace\n")
			continue
		}

		instrumentNamespace(ctx, client, ns.Name, excludedApps, orchestrator)
	}
}

func instrumentNamespace(ctx context.Context, client *kube.Client, ns string, excludedApps map[string]struct{}, orchestrator *lifecycle.Orchestrator) {
	deps, err := client.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Printf("  - \033[31mERROR\033[0m Cannot list deployments: %s\n", err)
		return
	}

	for _, dep := range deps.Items {
		fmt.Printf("  - Inspecting Deployment: %s\n", dep.Name)
		_, excluded := excludedApps[dep.Name]
		if excluded {
			fmt.Printf("    - Skipping deployment due to exclusion file\n")
			continue
		}

		orchestrator.Apply(ctx, &dep, &dep.Spec.Template)
	}
}

func runPreflightChecks(ctx context.Context, cmd *cobra.Command, client *kube.Client) {
	shouldSkip := cmd.Flag(skipPreflightCheckFlag).Changed && cmd.Flag(skipPreflightCheckFlag).Value.String() == "true"
	if shouldSkip {
		fmt.Printf("Skipping preflight checks due to --%s flag\n", skipPreflightCheckFlag)
		return
	}

	fmt.Printf("Running preflight checks:\n")
	for _, check := range preflight.AllChecks {
		fmt.Printf("  - %-60s", check.Description())
		if err := check.Execute(client, ctx); err != nil {
			fmt.Printf("\u001B[31mERROR\u001B[0m\n\n")
			fmt.Printf("Check failed: %s\n", err)
			os.Exit(1)
		} else {
			fmt.Printf("\u001B[32mPASS\u001B[0m\n")
		}
	}

	fmt.Printf("  - All preflight checks passed!\n\n")
}

func readFileLines(filePath string) (map[string]struct{}, error) {
	if filePath == "" {
		return nil, nil
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	lines := make(map[string]struct{})
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines[scanner.Text()] = struct{}{}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

func init() {
	rootCmd.AddCommand(instrumentCmd)
	instrumentCmd.AddCommand(clusterCmd)
	clusterCmd.Flags().String(excludeNamespacesFileFlag, "", "File containing namespaces to exclude from instrumentation. Each namespace should be on a new line.")
	clusterCmd.Flags().String(excludeAppsFileFlag, "", "File containing applications to exclude from instrumentation. Each application should be on a new line.")
	clusterCmd.Flags().Bool(skipPreflightCheckFlag, false, "Skip preflight checks")
}

func sliceToMap(slice []string) map[string]struct{} {
	m := make(map[string]struct{})
	for _, s := range slice {
		m[s] = struct{}{}
	}
	return m
}

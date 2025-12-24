package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/cmd/resources"
	cmdcontext "github.com/odigos-io/odigos/cli/pkg/cmd_context"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/cli/pkg/lifecycle"
	"github.com/odigos-io/odigos/cli/pkg/preflight"
	"github.com/odigos-io/odigos/cli/pkg/remote"
	"k8s.io/apimachinery/pkg/util/version"

	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func enableClusterSource(cmd *cobra.Command) {
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	var uiClient *remote.UIClientViaPortForward
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	defer func() {
		if uiClient != nil {
			uiClient.Close()
		}
	}()
	defer signal.Stop(ch)

	go func() {
		<-ch
		cancel()
		if uiClient != nil {
			uiClient.Close()
		}
	}()

	excludeNamespaces, err := readLinesFromFile(sourceExcludeNamespacesFileFlag)
	if err != nil {
		fmt.Printf("\033[31mERROR\033[0m Cannot read exclude namespaces file: %+v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Excluding %d namespaces\n", len(excludeNamespaces))

	excludeApps, err := readLinesFromFile(sourceExcludeWorkloadsFileFlag)
	if err != nil {
		fmt.Printf("\033[31mERROR\033[0m Cannot read exclude apps file: %+v\n", err)
		os.Exit(1)
	}

	dryRun := cmd.Flag(sourceDryRunFlagName).Changed && cmd.Flag(sourceDryRunFlagName).Value.String() == "true"
	isRemote := cmd.Flag(sourceRemoteFlagName).Changed && cmd.Flag(sourceRemoteFlagName).Value.String() == "true"
	coolOffStr := cmd.Flag(sourceInstrumentationCoolOffFlagName).Value.String()
	coolOff, err := time.ParseDuration(coolOffStr)
	if err != nil {
		fmt.Printf("\033[31mERROR\033[0m Invalid duration for instrumentation-cool-off: %s\n", err)
		os.Exit(1)
	}
	ctx = lifecycle.SetCoolOff(ctx, coolOff)

	onlyDeployment := cmd.Flag(sourceOnlyDeploymentFlagName).Value.String()
	onlyNamespace := cmd.Flag(sourceOnlyNamespaceFlagName).Value.String()

	if (onlyDeployment != "" && onlyNamespace == "") || (onlyDeployment == "" && onlyNamespace != "") {
		fmt.Printf("\033[31mERROR\033[0m --only-deployment and --only-namespace must be set together\n")
		os.Exit(1)
	}

	fmt.Printf("About to instrument with Odigos\n")
	if dryRun {
		fmt.Printf("Dry-Run mode ENABLED - No changes will be made\n")
	}

	fmt.Printf("%-50s", "Checking if Kubernetes cluster is reachable")
	client := kube.GetCLIClientOrExit(cmd)
	fmt.Printf("\u001B[32mPASS\u001B[0m\n\n")

	if isRemote {
		localPort := cmd.Flag(sourceLocalPortFlagName).Value.String()
		remotePort := cmd.Flag(sourceRemotePortFlagName).Value.String()
		localAddress := cmd.Flag(sourceLocalAddressFlagName).Value.String()

		ns, err := resources.GetOdigosNamespace(client, ctx)
		if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Cannot get odigos namespace: %s\n", err)
			os.Exit(1)
		}
		uiPod, err := kube.FindPodWithAppLabel(client, ctx, ns, k8sconsts.UIAppLabelValue)
		if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Cannot find odigos-ui pod: %s\n", err)
			os.Exit(1)
		}

		fw, err := kube.PortForwardWithContext(ctx, uiPod, client, localPort, remotePort, localAddress)
		if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Cannot start port-forward: %s\n", err)
			os.Exit(1)
		}

		uiClient, err = remote.NewUIClient(fw)
		if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Cannot create remote UI client: %s\n", err)
			os.Exit(1)
		}

		fmt.Println("Flag --remote is set, starting port-forward to UI pod ...")
		go func() {
			if err := uiClient.PortForwarder.ForwardPorts(); err != nil {
				fmt.Printf("\033[31mERROR\033[0m Cannot start remote UI client: %s\n", err)
				os.Exit(1)
			}
		}()

		// Wait for UI client to be ready with timeout
		select {
		case <-uiClient.PortForwarder.Ready:
			port, err := uiClient.DiscoverLocalPort()
			if err != nil {
				fmt.Printf("\033[31mERROR\033[0m Cannot discover local port for UI client: %s\n", err)
				os.Exit(1)
			}
			fmt.Printf("Remote client is using local port %s\n", port)
		case <-ctx.Done():
			fmt.Printf("\033[31mERROR\033[0m Context canceled while waiting for UI client to be ready\n")
			os.Exit(1)
		case <-time.After(30 * time.Second):
			fmt.Printf("\033[31mERROR\033[0m Timeout waiting for UI client to be ready\n")
			os.Exit(1)
		}
	}

	runPreflightChecks(ctx, cmd, client, isRemote)
	fmt.Printf("Starting instrumentation ...\n\n")
	instrumentCluster(ctx, client, excludeNamespaces, excludeApps, dryRun, isRemote, onlyNamespace, onlyDeployment)
}

func instrumentCluster(ctx context.Context, client *kube.Client, excludeNamespaces map[string]struct{}, excludeApps map[string]struct{}, dryRun bool, isRemote bool, onlyNamespace string, onlyDeployment string) {
	systemNs := sliceToMap(k8sconsts.DefaultIgnoredNamespaces)
	odigosNs, err := resources.GetOdigosNamespace(client, ctx)
	if err != nil {
		fmt.Printf("\033[31mERROR\033[0m Cannot get odigos namespace: %s\n", err)
		os.Exit(1)
	}
	systemNs[odigosNs] = struct{}{}

	orchestrator, err := lifecycle.NewOrchestrator(client, ctx, isRemote)
	if err != nil {
		fmt.Printf("\033[31mERROR\033[0m Cannot create orchestrator: %s\n", err)
		os.Exit(1)
	}
	if onlyDeployment != "" {
		dep, err := client.AppsV1().Deployments(onlyNamespace).Get(ctx, onlyDeployment, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Cannot get deployment %s in namespace %s: %s\n", onlyDeployment, onlyNamespace, err)
			os.Exit(1)
		}

		if dryRun {
			fmt.Printf("Dry-Run mode ENABLED - No changes will be made\n")
			return
		}

		err = orchestrator.Apply(ctx, dep)
		if err != nil {
			fmt.Printf("\033[31mERROR\033[0m Failed to instrument deployment: %s\n", err)
			os.Exit(1)
		}
		return
	}

	nsList, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Printf("\033[31mERROR\033[0m Cannot list namespaces: %s\n", err)
		os.Exit(1)
	}

	for _, ns := range nsList.Items {
		fmt.Printf("Instrumenting namespace: %s\n", ns.Name)
		excluded := isNamespaceExcluded(ns.Name, excludeNamespaces)
		_, system := systemNs[ns.Name]
		if excluded || system {
			fmt.Printf("  - Skipping namespace due to exclusion file or system namespace\n")
			continue
		}

		err = instrumentNamespace(ctx, client, ns.Name, excludeApps, orchestrator, dryRun)
		if errors.Is(err, context.Canceled) {
			return
		}
	}
}

func instrumentApp(ctx context.Context, app metav1.Object, excludedApps map[string]struct{}, orchestrator *lifecycle.Orchestrator, dryRun bool, kind string) error {
	fmt.Printf("  - Inspecting %s: %s\n", kind, app.GetName())
	if isAppExcluded(app, excludedApps, kind) {
		fmt.Printf("    - Skipping %s due to exclusion file\n", kind)
		return nil
	}
	if dryRun {
		fmt.Printf("    - Dry-Run mode ENABLED - No changes will be made\n")
		return nil
	}
	err := orchestrator.Apply(ctx, app)
	return err
}

// isAppExcluded checks if an app should be excluded based on the exclusion list.
// It checks three formats in order of specificity:
//  1. <namespace>/<kind>/<name> - most specific
//  2. <kind>/<name> - without namespace
//  3. <name> - just the name
//
// Kind matching is case-insensitive.
func isAppExcluded(app metav1.Object, excludedApps map[string]struct{}, kind string) bool {
	if excludedApps == nil {
		return false
	}

	name := app.GetName()
	namespace := app.GetNamespace()

	// Normalize kind string by removing any version suffixes like " (v1beta1)" or " (v1)"
	// This ensures "CronJob (v1)" matches "CronJob" in exclusion file
	normalizedKind := kind
	if idx := strings.Index(kind, " ("); idx != -1 {
		normalizedKind = kind[:idx]
	}

	// Convert kind to lowercase for case-insensitive matching
	lowerKind := strings.ToLower(normalizedKind)

	// Check all possible patterns with case-insensitive kind matching
	for excludedPattern := range excludedApps {
		// Check format: <namespace>/<kind>/<name>
		parts := strings.Split(excludedPattern, "/")
		if len(parts) == 3 {
			if parts[0] == namespace && strings.ToLower(parts[1]) == lowerKind && parts[2] == name {
				return true
			}
		}

		// Check format: <kind>/<name>
		if len(parts) == 2 {
			if strings.ToLower(parts[0]) == lowerKind && parts[1] == name {
				return true
			}
		}

		// Check format: <name>
		if len(parts) == 1 {
			if parts[0] == name {
				return true
			}
		}
	}

	return false
}

func instrumentNamespace(ctx context.Context, client *kube.Client, ns string, excludedApps map[string]struct{}, orchestrator *lifecycle.Orchestrator, dryRun bool) error {
	deps, err := client.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Printf("  - \033[31mERROR\033[0m Cannot list deployments: %s\n", err)
		return nil
	}
	for _, dep := range deps.Items {
		err = instrumentApp(ctx, &dep, excludedApps, orchestrator, dryRun, "Deployment")
		if isFatalError(err) {
			return err
		}
	}

	// StatefulSets
	statefulsets, err := client.AppsV1().StatefulSets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Printf("  - \033[31mERROR\033[0m Cannot list statefulsets: %s\n", err)
		return nil
	}
	for _, sts := range statefulsets.Items {
		err = instrumentApp(ctx, &sts, excludedApps, orchestrator, dryRun, "StatefulSet")
		if isFatalError(err) {
			return err
		}
	}

	// DaemonSets
	daemonsets, err := client.AppsV1().DaemonSets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Printf("  - \033[31mERROR\033[0m Cannot list daemonsets: %s\n", err)
		return nil
	}
	for _, ds := range daemonsets.Items {
		err = instrumentApp(ctx, &ds, excludedApps, orchestrator, dryRun, "DaemonSet")
		if isFatalError(err) {
			return err
		}
	}

	// CronJobs - handle both v1 and v1beta1
	ver := cmdcontext.K8SVersionFromContext(ctx)
	if ver.LessThan(version.MustParseSemantic("1.21.0")) {
		// Use v1beta1 for Kubernetes < 1.21
		cronjobs, err := client.BatchV1beta1().CronJobs(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			fmt.Printf("  - \033[31mERROR\033[0m Cannot list cronjobs (v1beta1): %s\n", err)
			return nil
		}
		for _, cj := range cronjobs.Items {
			err = instrumentApp(ctx, &cj, excludedApps, orchestrator, dryRun, "CronJob (v1beta1)")
			if isFatalError(err) {
				return err
			}
		}
	} else {
		// Use v1 for Kubernetes >= 1.21
		cronjobs, err := client.BatchV1().CronJobs(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			fmt.Printf("  - \033[31mERROR\033[0m Cannot list cronjobs (v1): %s\n", err)
			return nil
		}
		for _, cj := range cronjobs.Items {
			err = instrumentApp(ctx, &cj, excludedApps, orchestrator, dryRun, "CronJob (v1)")
			if isFatalError(err) {
				return err
			}
		}
	}

	return nil
}

// isFatalError checks for likely unrecoverable errors that should stop the instrumentation process.
func isFatalError(err error) bool {
	return errors.Is(err, context.Canceled) || apierrors.IsForbidden(err) || apierrors.IsUnauthorized(err)
}

func sliceToMap(slice []string) map[string]struct{} {
	m := make(map[string]struct{})
	for _, s := range slice {
		m[s] = struct{}{}
	}
	return m
}

func readLinesFromFile(filename string) (map[string]struct{}, error) {
	if filename == "" {
		return nil, nil
	}
	f, err := os.Open(filename)
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

// isNamespaceExcluded checks if a namespace matches any of the exclusion patterns.
// It supports both exact string matches and regex patterns.
func isNamespaceExcluded(namespace string, excludePatterns map[string]struct{}) bool {
	if excludePatterns == nil {
		return false
	}

	// First check for exact match
	if _, exists := excludePatterns[namespace]; exists {
		return true
	}

	// Then check if any pattern is a regex that matches
	for pattern := range excludePatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			// If it's not a valid regex, skip it (already checked for exact match above)
			continue
		}
		if re.MatchString(namespace) {
			return true
		}
	}

	return false
}

func runPreflightChecks(ctx context.Context, cmd *cobra.Command, client *kube.Client, remote bool) {
	shouldSkip := cmd.Flag(sourceSkipPreflightChecksFlagName).Changed && cmd.Flag(sourceSkipPreflightChecksFlagName).Value.String() == "true"
	if shouldSkip {
		fmt.Printf("Skipping preflight checks due to --%s flag\n", sourceSkipPreflightChecksFlagName)
		return
	}
	fmt.Printf("Running preflight checks:\n")
	for _, check := range preflight.AllChecks {
		fmt.Printf("  - %-60s", check.Description())
		if err := check.Execute(client, ctx, remote); err != nil {
			fmt.Printf("\u001B[31mERROR\u001B[0m\n\n")
			fmt.Printf("Check failed: %s\n", err)
			os.Exit(1)
		} else {
			fmt.Printf("\u001B[32mPASS\u001B[0m\n")
		}
	}
	fmt.Printf("  - All preflight checks passed!\n\n")
}

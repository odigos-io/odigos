package services

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	actionsapi "github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/common/urltemplate"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/scope"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/redis/go-redis/v9"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	unmatchedRedisKeyPrefixServer = "odigos:urltempl:unmatched:server"
	unmatchedRedisKeyPrefixClient = "odigos:urltempl:unmatched:client"
)

var (
	unmatchedRedisMu sync.Mutex
	unmatchedRedis   *redis.Client
)

func unmatchedRedisClient() (*redis.Client, error) {
	unmatchedRedisMu.Lock()
	defer unmatchedRedisMu.Unlock()
	if unmatchedRedis != nil {
		return unmatchedRedis, nil
	}
	ns := env.GetCurrentNamespace()
	addr := k8sconsts.OdigosCacheEndpoint(ns)
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("connect to cache at %s: %w", addr, err)
	}
	unmatchedRedis = rdb
	return unmatchedRedis, nil
}

func emptyLiveTrafficLearning() *model.URLTemplatizationLiveTrafficLearning {
	return &model.URLTemplatizationLiveTrafficLearning{
		Server:                 []*model.UnmatchedURLPath{},
		Client:                 []*model.UnmatchedURLPath{},
		ServerRecommendedRules: []*model.URLTemplatizationRecommendedRule{},
		ClientRecommendedRules: []*model.URLTemplatizationRecommendedRule{},
	}
}

// GetWorkloadUrlTemplatization returns the resolved UrlTemplatization config for a
// workload (existingConfigs) plus live-traffic learning data: unmatched HTTP path
// counts from cacheDb Redis and recommended rules computed on the fly from those
// counts (shared algorithm in common/urltemplate).
// Paths that already match an accepted URL templatization rule for the workload are
// omitted from both the path lists and the recommended-rule input.
// When live traffic learning is disabled, existingConfigs are still returned and
// liveTrafficLearning is empty.
func GetWorkloadUrlTemplatization(ctx context.Context, namespace, kind, name string) (*model.K8sWorkloadURLTemplatization, error) {
	existingConfigs, existingRules, err := loadExistingUrlTemplatizationFromIC(ctx, namespace, kind, name)
	if err != nil {
		return nil, err
	}

	cfg, err := getOdigosConfiguration(ctx)
	if err != nil {
		return nil, err
	}
	if cfg == nil || !common.UrlTemplatizationLiveTrafficLearningActive(cfg.CardinalityControl) {
		return &model.K8sWorkloadURLTemplatization{
			ExistingConfigs:     existingConfigs,
			LiveTrafficLearning: emptyLiveTrafficLearning(),
		}, nil
	}

	rdb, err := unmatchedRedisClient()
	if err != nil {
		return nil, err
	}

	workloadPrefix := fmt.Sprintf("%s/%s/%s/", namespace, kind, name)
	server, err := loadUnmatchedPaths(ctx, rdb, unmatchedRedisKeyPrefixServer, workloadPrefix)
	if err != nil {
		return nil, err
	}
	client, err := loadUnmatchedPaths(ctx, rdb, unmatchedRedisKeyPrefixClient, workloadPrefix)
	if err != nil {
		return nil, err
	}

	server = filterPathsNotMatchingRules(server, existingRules)
	client = filterPathsNotMatchingRules(client, existingRules)

	return &model.K8sWorkloadURLTemplatization{
		ExistingConfigs: existingConfigs,
		LiveTrafficLearning: &model.URLTemplatizationLiveTrafficLearning{
			Server:                 server,
			Client:                 client,
			ServerRecommendedRules: recommendedRulesFromPaths(server),
			ClientRecommendedRules: recommendedRulesFromPaths(client),
		},
	}, nil
}

// loadExistingUrlTemplatizationFromIC reads the resolved UrlTemplatization config from
// InstrumentationConfig (collector WorkloadCollectorConfig, else agent Traces).
// Returns GraphQL models and parsed PathRules used to filter already-covered paths.
func loadExistingUrlTemplatizationFromIC(ctx context.Context, namespace, kind, name string) ([]*model.URLTemplatizationExistingConfig, []urltemplate.PathRule, error) {
	if kube.DefaultClient == nil || kube.DefaultClient.OdigosClient == nil {
		return []*model.URLTemplatizationExistingConfig{}, nil, nil
	}

	icName := workload.CalculateWorkloadRuntimeObjectName(name, kind)
	ic, err := kube.DefaultClient.OdigosClient.InstrumentationConfigs(namespace).Get(ctx, icName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return []*model.URLTemplatizationExistingConfig{}, nil, nil
		}
		return nil, nil, fmt.Errorf("get instrumentationconfig %s/%s: %w", namespace, icName, err)
	}

	byContainer := collectUrlTemplatizationConfigsByContainer(ic)
	configs := make([]*model.URLTemplatizationExistingConfig, 0, len(byContainer))
	seenTemplates := make(map[string]struct{})
	var templates []string

	names := make([]string, 0, len(byContainer))
	for containerName := range byContainer {
		names = append(names, containerName)
	}
	sort.Strings(names)

	templatizationActions := listUrlTemplatizationActions(ctx)
	languages := containerLanguagesFromIC(ic)
	pw := k8sconsts.PodWorkload{Namespace: namespace, Kind: k8sconsts.WorkloadKind(kind), Name: name}

	for _, containerName := range names {
		cfg := byContainer[containerName]
		resolved := resolveExistingTemplates(templatizationActions, pw, languages[containerName], cfg)
		configs = append(configs, urlTemplatizationConfigToExistingModel(containerName, cfg, resolved))
		for _, item := range resolved {
			if _, ok := seenTemplates[item.Template]; ok {
				continue
			}
			seenTemplates[item.Template] = struct{}{}
			templates = append(templates, item.Template)
		}
	}

	return configs, parseTemplatizationPathRules(templates), nil
}

// listUrlTemplatizationActions returns the enabled URLTemplatization actions.
// Best effort: when actions cannot be listed, templates are still reported from
// the resolved config, just without their examples / notes / owning action.
func listUrlTemplatizationActions(ctx context.Context) []odigosv1.Action {
	actions, err := kube.DefaultClient.OdigosClient.Actions(env.GetCurrentNamespace()).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil
	}

	out := make([]odigosv1.Action, 0, len(actions.Items))
	for i := range actions.Items {
		action := actions.Items[i]
		if action.Spec.Disabled || action.Spec.URLTemplatization == nil {
			continue
		}
		out = append(out, action)
	}
	return out
}

func containerLanguagesFromIC(ic *odigosv1.InstrumentationConfig) map[string]common.ProgrammingLanguage {
	languages := make(map[string]common.ProgrammingLanguage, len(ic.Status.RuntimeDetailsByContainer))
	for _, details := range ic.Status.RuntimeDetailsByContainer {
		languages[details.ContainerName] = details.Language
	}
	return languages
}

// resolveExistingTemplates attributes each template applied to a container to the
// Action rule group it came from, by iterating the rule scopes the same way the
// instrumentor does when it resolves the container config.
// Templates in the resolved config that no longer match any rule are kept without
// an owning action, so nothing the collector applies is hidden from the user.
func resolveExistingTemplates(
	actions []odigosv1.Action,
	pw k8sconsts.PodWorkload,
	language common.ProgrammingLanguage,
	cfg *actionsapi.UrlTemplatizationConfig,
) []*model.URLTemplatizationExistingTemplate {
	resolved := make([]*model.URLTemplatizationExistingTemplate, 0, len(cfg.Templates))
	byTemplate := make(map[string]struct{}, len(cfg.Templates))

	add := func(item *model.URLTemplatizationExistingTemplate) {
		if item.Template == "" {
			return
		}
		if _, ok := byTemplate[item.Template]; ok {
			return
		}
		byTemplate[item.Template] = struct{}{}
		resolved = append(resolved, item)
	}

	for i := range actions {
		action := &actions[i]
		actionID := action.Name
		actionName := action.Spec.ActionName
		managedBy := managedByFromLabels(action.Labels)

		for _, rules := range action.Spec.URLTemplatization.Rules {
			if !scope.SourceScopeMatchesContainer(rules.Scopes, pw, language) {
				continue
			}

			// Documented templates come first so their examples / notes win over the
			// same template declared as a plain string.
			for _, documented := range rules.DocumentedTemplates {
				examples := documented.Examples
				if examples == nil {
					examples = []string{}
				}
				add(&model.URLTemplatizationExistingTemplate{
					Template:   strings.TrimSpace(documented.Template),
					Examples:   examples,
					Notes:      nullableString(documented.Notes),
					ActionID:   &actionID,
					ActionName: nullableString(actionName),
					ManagedBy:  managedBy,
				})
			}

			for _, template := range rules.Templates {
				add(&model.URLTemplatizationExistingTemplate{
					Template:   strings.TrimSpace(template),
					Examples:   []string{},
					ActionID:   &actionID,
					ActionName: nullableString(actionName),
					ManagedBy:  managedBy,
				})
			}
		}
	}

	for _, template := range cfg.Templates {
		add(&model.URLTemplatizationExistingTemplate{
			Template:  strings.TrimSpace(template),
			Examples:  []string{},
			ManagedBy: model.ManagedByUnknown,
		})
	}

	sort.Slice(resolved, func(i, j int) bool { return resolved[i].Template < resolved[j].Template })
	return resolved
}

func nullableString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func collectUrlTemplatizationConfigsByContainer(ic *odigosv1.InstrumentationConfig) map[string]*actionsapi.UrlTemplatizationConfig {
	byContainer := make(map[string]*actionsapi.UrlTemplatizationConfig)

	for i := range ic.Spec.WorkloadCollectorConfig {
		c := &ic.Spec.WorkloadCollectorConfig[i]
		if c.UrlTemplatization != nil {
			byContainer[c.ContainerName] = c.UrlTemplatization
		}
	}
	for i := range ic.Spec.Containers {
		c := &ic.Spec.Containers[i]
		if _, ok := byContainer[c.ContainerName]; ok {
			continue
		}
		if c.Traces != nil && c.Traces.UrlTemplatization != nil {
			byContainer[c.ContainerName] = c.Traces.UrlTemplatization
		}
	}
	return byContainer
}

func urlTemplatizationConfigToExistingModel(
	containerName string,
	cfg *actionsapi.UrlTemplatizationConfig,
	templates []*model.URLTemplatizationExistingTemplate,
) *model.URLTemplatizationExistingConfig {
	if templates == nil {
		templates = []*model.URLTemplatizationExistingTemplate{}
	}
	out := &model.URLTemplatizationExistingConfig{
		ContainerName: containerName,
		Templates:     templates,
	}
	if cfg.Default != nil {
		out.Default = &model.URLTemplatizationExistingDefault{
			Disabled: cfg.Default.Disabled,
		}
		if cfg.Default.SkipPolicy != nil {
			skipForNonSuccess := cfg.Default.SkipPolicy.SkipForNonSuccessCodes
			out.Default.SkipPolicy = &model.URLTemplatizationDefaultSkipPolicy{
				SkipForNonSuccessCodes: &skipForNonSuccess,
				SkipHTTPStatusCodes:    cfg.Default.SkipPolicy.SkipHttpStatusCodes,
			}
		}
	}
	return out
}

func parseTemplatizationPathRules(templates []string) []urltemplate.PathRule {
	rules := make([]urltemplate.PathRule, 0, len(templates))
	for _, template := range templates {
		rule, err := urltemplate.ParseUserInputRuleString(template, false)
		if err != nil {
			continue
		}
		rules = append(rules, rule)
	}
	return rules
}

func filterPathsNotMatchingRules(paths []*model.UnmatchedURLPath, rules []urltemplate.PathRule) []*model.UnmatchedURLPath {
	if len(paths) == 0 || len(rules) == 0 {
		return paths
	}
	out := make([]*model.UnmatchedURLPath, 0, len(paths))
	for _, item := range paths {
		if item == nil {
			continue
		}
		if pathMatchesAnyRule(item.Path, rules) {
			continue
		}
		out = append(out, item)
	}
	return out
}

func pathMatchesAnyRule(path string, rules []urltemplate.PathRule) bool {
	for _, rule := range rules {
		if rule.IsPathMatching(path) {
			return true
		}
	}
	return false
}

// RecommendLiveTrafficRulesFromCounts turns unmatched path Redis hash fields
// (path -> count string) into recommended URL templatization rules for the UI.
// Enterprise UI wires this to common-enterprise live-traffic learning; the OSS
// default returns no suggestions.
var RecommendLiveTrafficRulesFromCounts = func(pathCounts map[string]string) []*model.URLTemplatizationRecommendedRule {
	return nil
}

func recommendedRulesFromPaths(paths []*model.UnmatchedURLPath) []*model.URLTemplatizationRecommendedRule {
	aggregated := make(map[string]int64, len(paths))
	for _, item := range paths {
		if item == nil || item.Path == "" {
			continue
		}
		aggregated[item.Path] += int64(item.Count)
	}
	pathCounts := make(map[string]string, len(aggregated))
	for path, count := range aggregated {
		pathCounts[path] = strconv.FormatInt(count, 10)
	}
	return RecommendLiveTrafficRulesFromCounts(pathCounts)
}

func loadUnmatchedPaths(ctx context.Context, rdb *redis.Client, kindPrefix, workloadPrefix string) ([]*model.UnmatchedURLPath, error) {
	pattern := fmt.Sprintf("%s:%s*", kindPrefix, workloadPrefix)
	var cursor uint64
	aggregated := map[string]*model.UnmatchedURLPath{}

	for {
		keys, next, err := rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("scan unmatched paths (%s): %w", pattern, err)
		}
		for _, key := range keys {
			containerName := containerFromUnmatchedKey(key, kindPrefix, workloadPrefix)
			fields, err := rdb.HGetAll(ctx, key).Result()
			if err != nil {
				return nil, fmt.Errorf("hgetall %s: %w", key, err)
			}
			for path, countStr := range fields {
				count, err := strconv.ParseInt(countStr, 10, 64)
				if err != nil {
					continue
				}
				aggKey := path + "\x00" + containerName
				if existing, ok := aggregated[aggKey]; ok {
					existing.Count += int(count)
					continue
				}
				cn := containerName
				aggregated[aggKey] = &model.UnmatchedURLPath{
					Path:          path,
					Count:         int(count),
					ContainerName: &cn,
				}
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}

	out := make([]*model.UnmatchedURLPath, 0, len(aggregated))
	for _, item := range aggregated {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Path < out[j].Path
	})
	return out, nil
}

func containerFromUnmatchedKey(key, kindPrefix, workloadPrefix string) string {
	// key = odigos:urltempl:unmatched:{server|client}:{ns}/{kind}/{name}/{container}
	prefix := kindPrefix + ":" + workloadPrefix
	if !strings.HasPrefix(key, prefix) {
		return ""
	}
	return strings.TrimPrefix(key, prefix)
}

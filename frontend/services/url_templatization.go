package services

import (
	"context"
	"fmt"
	"sort"
	"strings"

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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetContainerUrlTemplatization resolves URL templatization for one container from
// InstrumentationConfig + matching URLTemplatization Actions. Also returns parsed
// path rules for callers that filter unmatched live-traffic paths.
func GetContainerUrlTemplatization(ctx context.Context, namespace, kind, name, containerName string) (*model.LiveTrafficLearningTemplatization, []urltemplate.PathRule, error) {
	if kube.DefaultClient == nil || kube.DefaultClient.OdigosClient == nil {
		return &model.LiveTrafficLearningTemplatization{Templates: []*model.LiveTrafficLearningTemplate{}}, nil, nil
	}

	icName := workload.CalculateWorkloadRuntimeObjectName(name, kind)
	ic, err := kube.DefaultClient.OdigosClient.InstrumentationConfigs(namespace).Get(ctx, icName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return &model.LiveTrafficLearningTemplatization{Templates: []*model.LiveTrafficLearningTemplate{}}, nil, nil
		}
		return nil, nil, fmt.Errorf("get instrumentationconfig %s/%s: %w", namespace, icName, err)
	}

	byContainer := collectUrlTemplatizationConfigsByContainer(ic)
	cfg := byContainer[containerName]
	if cfg == nil {
		return &model.LiveTrafficLearningTemplatization{Templates: []*model.LiveTrafficLearningTemplate{}}, nil, nil
	}

	languages := containerLanguagesFromIC(ic)
	pw := k8sconsts.PodWorkload{Namespace: namespace, Kind: k8sconsts.WorkloadKind(kind), Name: name}
	resolved := ResolveExistingUrlTemplatizationTemplates(listUrlTemplatizationActions(ctx), pw, languages[containerName], cfg)
	templatization := UrlTemplatizationConfigToModel(cfg, resolved)

	templates := make([]string, 0, len(resolved))
	for _, item := range resolved {
		templates = append(templates, item.Template)
	}
	return templatization, ParseTemplatizationPathRules(templates), nil
}

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

// ResolveExistingUrlTemplatizationTemplates merges Action-documented templates with
// templates present on the resolved InstrumentationConfig for the container.
func ResolveExistingUrlTemplatizationTemplates(
	actions []odigosv1.Action,
	pw k8sconsts.PodWorkload,
	language common.ProgrammingLanguage,
	cfg *actionsapi.UrlTemplatizationConfig,
) []*model.LiveTrafficLearningTemplate {
	resolved := make([]*model.LiveTrafficLearningTemplate, 0, len(cfg.Templates))
	byTemplate := make(map[string]struct{}, len(cfg.Templates))

	add := func(item *model.LiveTrafficLearningTemplate) {
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
		managedBy := ManagedByFromLabels(action.Labels)

		for _, rules := range action.Spec.URLTemplatization.Rules {
			if !scope.SourceScopeMatchesContainer(rules.Scopes, pw, language) {
				continue
			}

			for _, documented := range rules.DocumentedTemplates {
				examples := documented.Examples
				if examples == nil {
					examples = []string{}
				}
				add(&model.LiveTrafficLearningTemplate{
					Template:   strings.TrimSpace(documented.Template),
					Examples:   examples,
					Notes:      nullableString(documented.Notes),
					ActionID:   &actionID,
					ActionName: nullableString(actionName),
					ManagedBy:  managedBy,
				})
			}

			for _, template := range rules.Templates {
				add(&model.LiveTrafficLearningTemplate{
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
		add(&model.LiveTrafficLearningTemplate{
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

// UrlTemplatizationConfigToModel maps resolved templates + pattern-based defaults
// into the GraphQL LiveTrafficLearningTemplatization shape.
func UrlTemplatizationConfigToModel(
	cfg *actionsapi.UrlTemplatizationConfig,
	templates []*model.LiveTrafficLearningTemplate,
) *model.LiveTrafficLearningTemplatization {
	if templates == nil {
		templates = []*model.LiveTrafficLearningTemplate{}
	}
	out := &model.LiveTrafficLearningTemplatization{
		Templates: templates,
	}
	if cfg.Default != nil {
		out.PatternBasedTemplatization = &model.LiveTrafficLearningPatternBasedTemplatization{
			Disabled: cfg.Default.Disabled,
		}
		if cfg.Default.SkipPolicy != nil {
			skipForNonSuccess := cfg.Default.SkipPolicy.SkipForNonSuccessCodes
			out.PatternBasedTemplatization.SkipPolicy = &model.URLTemplatizationDefaultSkipPolicy{
				SkipForNonSuccessCodes: &skipForNonSuccess,
				SkipHTTPStatusCodes:    cfg.Default.SkipPolicy.SkipHttpStatusCodes,
			}
		}
	}
	return out
}

// ParseTemplatizationPathRules parses template strings into path matchers.
func ParseTemplatizationPathRules(templates []string) []urltemplate.PathRule {
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

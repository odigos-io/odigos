package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"

	"github.com/odigos-io/odigos/status"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//go:embed file.go.tpl
var fileTemplate string

type reasonData struct {
	Name               string
	Title              string
	Summary            string
	Description        string
	Message            string
	State              string
	K8sConditionStatus metav1.ConditionStatus
	OdigosSeverity     status.OdigosSeverity
	ActionItems        []actionItemData
}

type actionItemData struct {
	Type       status.ActionItemType
	ButtonText string
}

type docsData struct {
	Title       string
	Summary     string
	Description string
	States      []stateDocData
}

type stateDocData struct {
	State   string
	Summary string
}

type parameterData struct {
	Name        string
	Description string
}

type fileData struct {
	Package               string
	TypeName              string
	OwnerResource         string
	Scope                 string
	Component             string
	HasDocs               bool
	Docs                  docsData
	Parameters            []parameterData
	Reasons               []reasonData
	HasK8sConditionStatus bool
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "generate" || d.Name() == "generated" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".yaml") && !strings.HasSuffix(d.Name(), ".yml") {
			return nil
		}
		return generateFile(path)
	})
	if err != nil {
		fatal(err)
	}
}

// messageParamsYAML is generator-local: parameters are documentation for code
// generation only and are not part of the runtime status model.
type messageParamsYAML struct {
	Spec struct {
		Parameters []struct {
			Name        string `yaml:"name"`
			Description string `yaml:"description"`
		} `yaml:"parameters"`
	} `yaml:"spec"`
}

func generateFile(yamlPath string) error {
	bytesData, err := os.ReadFile(yamlPath)
	if err != nil {
		return err
	}

	var s status.OdigosStatus
	if err := yaml.Unmarshal(bytesData, &s); err != nil {
		return fmt.Errorf("unmarshal %s: %w", yamlPath, err)
	}

	var messageParams messageParamsYAML
	if err := yaml.Unmarshal(bytesData, &messageParams); err != nil {
		return fmt.Errorf("unmarshal parameters from %s: %w", yamlPath, err)
	}

	data := fileData{
		Package:       "generated",
		TypeName:      s.Spec.Type,
		OwnerResource: s.Metadata.OwnerResource,
		Scope:         s.Metadata.Scope,
		Component:     s.Metadata.Component,
	}

	if s.Spec.Docs.Title != "" || s.Spec.Docs.Summary != "" || s.Spec.Docs.Description != "" || len(s.Spec.Docs.States) > 0 {
		data.HasDocs = true
		docs := docsData{
			Title:       s.Spec.Docs.Title,
			Summary:     s.Spec.Docs.Summary,
			Description: s.Spec.Docs.Description,
		}
		for _, state := range s.Spec.Docs.States {
			docs.States = append(docs.States, stateDocData{
				State:   state.State,
				Summary: state.Summary,
			})
		}
		data.Docs = docs
	}

	for _, param := range messageParams.Spec.Parameters {
		if param.Name == "" {
			return fmt.Errorf("%s: parameter with empty name", yamlPath)
		}
		data.Parameters = append(data.Parameters, parameterData{
			Name:        param.Name,
			Description: strings.TrimSpace(param.Description),
		})
	}

	for _, reason := range s.Spec.Reasons {
		expanded := expandReasonStates(reason)
		for _, rd := range expanded {
			if rd.K8sConditionStatus != "" {
				data.HasK8sConditionStatus = true
			}
			data.Reasons = append(data.Reasons, rd)
		}
	}

	tmpl, err := template.New("status").Funcs(template.FuncMap{
		"k8sConditionStatusConst": k8sConditionStatusConst,
		"odigosSeverityConst":     odigosSeverityConst,
		"actionItemTypeConst":     actionItemTypeConst,
		"splitLines":              splitLines,
	}).Parse(fileTemplate)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("format generated code for %s: %w\n%s", yamlPath, err, buf.String())
	}

	baseName := strings.TrimSuffix(filepath.Base(yamlPath), filepath.Ext(yamlPath))
	outDir := filepath.Join(filepath.Dir(yamlPath), "generated")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	outPath := filepath.Join(outDir, baseName+".go")
	if err := os.WriteFile(outPath, formatted, 0o644); err != nil {
		return err
	}

	fmt.Printf("generated %s\n", outPath)
	return nil
}

// expandReasonStates turns a YAML reason into one generated reason per state.
// Reasons without states produce a single reason using the reason-level fields.
func expandReasonStates(reason status.Reason) []reasonData {
	actionItems := make([]actionItemData, 0, len(reason.ActionItems))
	for _, actionItem := range reason.ActionItems {
		actionItems = append(actionItems, actionItemData{
			Type:       actionItem.Type,
			ButtonText: actionItem.ButtonText,
		})
	}

	if len(reason.States) == 0 {
		return []reasonData{{
			Name:               reason.Name,
			Title:              reason.Title,
			Summary:            reason.Summary,
			Description:        reason.Description,
			Message:            reason.Message,
			K8sConditionStatus: reason.K8sConditionStatus,
			OdigosSeverity:     reason.OdigosSeverity,
			ActionItems:        actionItems,
		}}
	}

	out := make([]reasonData, 0, len(reason.States))
	for _, state := range reason.States {
		title := state.Title
		if title == "" {
			title = reason.Title
		}
		summary := state.Summary
		if summary == "" {
			summary = reason.Summary
		}
		message := state.Message
		if message == "" {
			message = reason.Message
		}

		out = append(out, reasonData{
			Name:               reason.Name + "_" + capitalize(state.State),
			Title:              title,
			Summary:            summary,
			Description:        reason.Description,
			Message:            message,
			State:              state.State,
			K8sConditionStatus: reason.K8sConditionStatus,
			OdigosSeverity:     reason.OdigosSeverity,
			ActionItems:        actionItems,
		})
	}
	return out
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func odigosSeverityConst(severity status.OdigosSeverity) string {
	switch severity {
	case status.OdigosSeverityError:
		return "status.OdigosSeverityError"
	case status.OdigosSeverityFailure:
		return "status.OdigosSeverityFailure"
	case status.OdigosSeverityNotice:
		return "status.OdigosSeverityNotice"
	case status.OdigosSeverityPending:
		return "status.OdigosSeverityPending"
	case status.OdigosSeverityWaiting:
		return "status.OdigosSeverityWaiting"
	case status.OdigosSeverityUnsupported:
		return "status.OdigosSeverityUnsupported"
	case status.OdigosSeverityDisabled:
		return "status.OdigosSeverityDisabled"
	case status.OdigosSeveritySuccess:
		return "status.OdigosSeveritySuccess"
	case status.OdigosSeverityIrrelevant:
		return "status.OdigosSeverityIrrelevant"
	case status.OdigosSeverityUnknown:
		return "status.OdigosSeverityUnknown"
	default:
		fatal(fmt.Errorf("unknown odigosSeverity %q", severity))
		return ""
	}
}

func k8sConditionStatusConst(condStatus metav1.ConditionStatus) string {
	switch condStatus {
	case metav1.ConditionTrue:
		return "metav1.ConditionTrue"
	case metav1.ConditionFalse:
		return "metav1.ConditionFalse"
	case metav1.ConditionUnknown:
		return "metav1.ConditionUnknown"
	default:
		fatal(fmt.Errorf("unknown k8sConditionStatus %q", condStatus))
		return ""
	}
}

func actionItemTypeConst(actionType status.ActionItemType) string {
	switch actionType {
	case status.ActionItemTypeRolloutWorkload:
		return "status.ActionItemTypeRolloutWorkload"
	default:
		fatal(fmt.Errorf("unknown action item type %q", actionType))
		return ""
	}
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "status generate: %v\n", err)
	os.Exit(1)
}

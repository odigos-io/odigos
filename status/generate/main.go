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

	"github.com/odigos-io/odigos/status"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//go:embed file.go.tpl
var fileTemplate string

type reasonData struct {
	Name                 string
	Message              string
	TechnicalDescription string
	K8sConditionStatus   metav1.ConditionStatus
	OdigosSeverity       status.OdigosSeverity
	ActionItems          []actionItemData
}

type actionItemData struct {
	Type           status.ActionItemType
	UserFacingText string
}

type fileData struct {
	Package               string
	TypeName              string
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

func generateFile(yamlPath string) error {
	bytesData, err := os.ReadFile(yamlPath)
	if err != nil {
		return err
	}

	var s status.OdigosStatus
	if err := yaml.Unmarshal(bytesData, &s); err != nil {
		return fmt.Errorf("unmarshal %s: %w", yamlPath, err)
	}

	data := fileData{
		Package:  "generated",
		TypeName: s.Spec.Type,
	}

	for _, reason := range s.Spec.Reasons {
		rd := reasonData{
			Name:                 reason.Name,
			Message:              escapeString(reason.Message),
			TechnicalDescription: escapeString(reason.TechnicalDescription),
			OdigosSeverity:       reason.OdigosSeverity,
		}
		for _, actionItem := range reason.ActionItems {
			rd.ActionItems = append(rd.ActionItems, actionItemData{
				Type:           actionItem.Type,
				UserFacingText: escapeString(actionItem.UserFacingText),
			})
		}
		if reason.K8sConditionStatus != "" {
			rd.K8sConditionStatus = reason.K8sConditionStatus
			data.HasK8sConditionStatus = true
		}
		data.Reasons = append(data.Reasons, rd)
	}

	tmpl, err := template.New("status").Funcs(template.FuncMap{
		"k8sConditionStatusConst": k8sConditionStatusConst,
		"odigosSeverityConst":     odigosSeverityConst,
		"actionItemTypeConst":     actionItemTypeConst,
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

func escapeString(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "status generate: %v\n", err)
	os.Exit(1)
}

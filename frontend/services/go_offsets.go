package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetGoOffsets returns parsed go_offset_results.json from the
// odigos-go-offsets ConfigMap in the Odigos namespace.
func GetGoOffsets(ctx context.Context) (*model.GoOffsets, error) {
	content, err := getGoOffsetsRaw(ctx)
	if err != nil {
		return nil, err
	}

	parsed, err := parseGoOffsetsContent(content)
	if err != nil {
		return nil, err
	}

	return goOffsetsToModel(parsed), nil
}

func getGoOffsetsRaw(ctx context.Context) (string, error) {
	ns := env.GetCurrentNamespace()

	var cm corev1.ConfigMap
	err := kube.CacheClient.Get(ctx, client.ObjectKey{
		Namespace: ns,
		Name:      k8sconsts.GoOffsetsConfigMap,
	}, &cm)
	if err != nil {
		return "", fmt.Errorf("failed to get ConfigMap %s/%s: %w", ns, k8sconsts.GoOffsetsConfigMap, err)
	}

	content, ok := cm.Data[k8sconsts.GoOffsetsFileName]
	if !ok {
		return "", fmt.Errorf("key %q not found in ConfigMap %s/%s", k8sconsts.GoOffsetsFileName, ns, k8sconsts.GoOffsetsConfigMap)
	}

	return content, nil
}

func parseGoOffsetsContent(content string) (*versionedModules, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return &versionedModules{Mods: []*jsonModule{}}, nil
	}

	// ConfigMap data is a JSON-encoded string (see odigos pro update-offsets / Helm).
	var inner string
	if err := json.Unmarshal([]byte(trimmed), &inner); err != nil {
		return nil, fmt.Errorf("invalid go offsets JSON: %w", err)
	}
	return parseGoOffsetsFile(inner)
}

// parseGoOffsetsFile parses offsets file content as returned by the
// public offsets URL (raw JSON, optional ---SIGNATURE--- trailer).
func parseGoOffsetsFile(content string) (*versionedModules, error) {
	inner := strings.TrimSpace(content)
	if inner == "" {
		return &versionedModules{Mods: []*jsonModule{}}, nil
	}
	inner = stripGoOffsetsSignature(inner)
	if inner == "" {
		return &versionedModules{Mods: []*jsonModule{}}, nil
	}

	var parsed versionedModules
	if err := json.Unmarshal([]byte(inner), &parsed); err != nil {
		return nil, fmt.Errorf("invalid go offsets JSON: %w", err)
	}

	if parsed.Mods == nil {
		parsed.Mods = []*jsonModule{}
	}

	return &parsed, nil
}

const goOffsetsSignatureDelimiter = "---SIGNATURE---"

func stripGoOffsetsSignature(inner string) string {
	parts := strings.SplitN(inner, goOffsetsSignatureDelimiter, 2)
	return strings.TrimSpace(parts[0])
}

func goOffsetsToModel(parsed *versionedModules) *model.GoOffsets {
	mods := make([]*model.GoOffsetModule, 0, len(parsed.Mods))
	for _, mod := range parsed.Mods {
		if mod == nil {
			continue
		}
		byMinorVersions, minVersion, maxVersion := moduleVersions(mod)
		minorVersionsModel := make([]*model.GoOffsetMinorVersion, 0, len(byMinorVersions))
		minorVersions := make([]string, 0, len(byMinorVersions))
		for majorMinor := range byMinorVersions {
			minorVersions = append(minorVersions, majorMinor)
		}
		sort.Slice(minorVersions, func(i, j int) bool {
			return compareVersions(minorVersions[j], minorVersions[i])
		})

		for _, minorVersion := range minorVersions {
			versionsForThisMinor := byMinorVersions[minorVersion]
			sort.Slice(versionsForThisMinor, func(i, j int) bool {
				return compareVersions(versionsForThisMinor[i], versionsForThisMinor[j])
			})
			minorVersionsModel = append(minorVersionsModel, &model.GoOffsetMinorVersion{
				MinorVersion: minorVersion,
				Versions:     versionsForThisMinor,
			})
		}
		mods = append(mods, &model.GoOffsetModule{
			Module:        mod.Module,
			MinVersion:    minVersion,
			MaxVersion:    maxVersion,
			MinorVersions: minorVersionsModel,
		})
	}

	sort.Slice(mods, func(i, j int) bool {
		return strings.ToLower(mods[i].Module) < strings.ToLower(mods[j].Module)
	})

	timestamp := ""
	if !parsed.Timestamp.IsZero() {
		timestamp = parsed.Timestamp.UTC().Format(time.RFC3339Nano)
	}

	return &model.GoOffsets{
		Timestamp: timestamp,
		Mods:      mods,
	}
}

func moduleVersions(mod *jsonModule) (byMinorVersions map[string][]string, minVersion, maxVersion string) {
	seen := make(map[string]struct{})

	var min, max *version.Version
	var minorToVersions map[string][]string

	consider := func(raw string) {
		if _, ok := seen[raw]; ok {
			return
		}
		v, err := version.NewVersion(raw)
		if err != nil {
			return
		}
		seen[raw] = struct{}{}
		if min == nil || v.LessThan(min) {
			min = v
		}
		if max == nil || v.GreaterThan(max) {
			max = v
		}
		seg := v.Segments()
		majorMinor := fmt.Sprintf("%d.%d", seg[0], seg[1])
		if minorToVersions == nil {
			minorToVersions = make(map[string][]string)
		}
		minorToVersions[majorMinor] = append(minorToVersions[majorMinor], raw)
	}

	for _, pkg := range mod.Packages {
		if pkg == nil {
			continue
		}
		for _, s := range pkg.Structs {
			if s == nil {
				continue
			}
			for _, f := range s.Fields {
				if f == nil {
					continue
				}
				for _, o := range f.Offsets {
					if o == nil {
						continue
					}
					for _, v := range o.Versions {
						consider(v)
					}
				}
			}
		}
	}

	if min == nil || max == nil {
		return minorToVersions, "", ""
	}
	return minorToVersions, min.String(), max.String()
}

func compareVersions(a, b string) bool {
	va, errA := version.NewVersion(a)
	vb, errB := version.NewVersion(b)
	if errA != nil || errB != nil {
		return a < b
	}
	return va.LessThan(vb)
}

// CheckGoOffsetsUpdates compares candidate offsets file content
// against the installed ConfigMap without writing. The result is the current
// offsets plus versions/modules that would be added by an update.
func CheckGoOffsetsUpdates(ctx context.Context, content string) (*model.GoOffsetsUpdateCheck, error) {
	currentRaw, err := getGoOffsetsRaw(ctx)
	if err != nil {
		return nil, err
	}
	current, err := parseGoOffsetsContent(currentRaw)
	if err != nil {
		return nil, err
	}
	proposed, err := parseGoOffsetsFile(content)
	if err != nil {
		return nil, err
	}
	return compareGoOffsets(current, proposed), nil
}

func compareGoOffsets(current, proposed *versionedModules) *model.GoOffsetsUpdateCheck {
	currentVersions := make(map[string]map[string]struct{}, len(current.Mods))
	moduleNames := make([]string, 0, len(current.Mods)+len(proposed.Mods))
	seenModules := make(map[string]struct{}, len(current.Mods)+len(proposed.Mods))

	for _, mod := range current.Mods {
		if mod == nil || mod.Module == "" {
			continue
		}
		currentVersions[mod.Module] = moduleVersionSet(mod)
		if _, ok := seenModules[mod.Module]; !ok {
			seenModules[mod.Module] = struct{}{}
			moduleNames = append(moduleNames, mod.Module)
		}
	}

	proposedVersions := make(map[string]map[string]struct{}, len(proposed.Mods))
	for _, mod := range proposed.Mods {
		if mod == nil || mod.Module == "" {
			continue
		}
		proposedVersions[mod.Module] = moduleVersionSet(mod)
		if _, ok := seenModules[mod.Module]; !ok {
			seenModules[mod.Module] = struct{}{}
			moduleNames = append(moduleNames, mod.Module)
		}
	}
	sort.Slice(moduleNames, func(i, j int) bool {
		return strings.ToLower(moduleNames[i]) < strings.ToLower(moduleNames[j])
	})

	hasUpdates := false
	mods := make([]*model.GoOffsetModuleUpdate, 0, len(moduleNames))
	for _, name := range moduleNames {
		curSet, existsInCurrent := currentVersions[name]
		propSet := proposedVersions[name]
		isNewModule := !existsInCurrent
		// An update writes the candidate file over the ConfigMap, so a module
		// the candidate omits is dropped rather than left alone.
		isRemovedModule := existsInCurrent && len(propSet) == 0
		if (isNewModule && len(propSet) > 0) || isRemovedModule {
			hasUpdates = true
		}

		allVersions := make([]string, 0, len(curSet)+len(propSet))
		newVersionSet := make(map[string]struct{})
		removedVersionSet := make(map[string]struct{})
		for v := range curSet {
			allVersions = append(allVersions, v)
			if _, ok := propSet[v]; !ok {
				removedVersionSet[v] = struct{}{}
				hasUpdates = true
			}
		}
		for v := range propSet {
			if _, ok := curSet[v]; ok {
				continue
			}
			allVersions = append(allVersions, v)
			newVersionSet[v] = struct{}{}
			hasUpdates = true
		}

		byMinor := make(map[string][]string)
		var min, max *version.Version
		for _, raw := range allVersions {
			v, err := version.NewVersion(raw)
			if err != nil {
				continue
			}
			// The range describes what an update would leave in place, so versions
			// it drops are listed but excluded. A module dropped in full has no
			// post-update range, so it keeps its installed one.
			_, dropped := removedVersionSet[raw]
			if !dropped || isRemovedModule {
				if min == nil || v.LessThan(min) {
					min = v
				}
				if max == nil || v.GreaterThan(max) {
					max = v
				}
			}
			seg := v.Segments()
			majorMinor := fmt.Sprintf("%d.%d", seg[0], seg[1])
			byMinor[majorMinor] = append(byMinor[majorMinor], raw)
		}

		minorKeys := make([]string, 0, len(byMinor))
		for k := range byMinor {
			minorKeys = append(minorKeys, k)
		}
		sort.Slice(minorKeys, func(i, j int) bool {
			return compareVersions(minorKeys[j], minorKeys[i])
		})

		minorVersions := make([]*model.GoOffsetMinorVersionUpdate, 0, len(minorKeys))
		for _, minorKey := range minorKeys {
			versions := byMinor[minorKey]
			sort.Slice(versions, func(i, j int) bool {
				return compareVersions(versions[i], versions[j])
			})
			versionUpdates := make([]*model.GoOffsetVersionUpdate, 0, len(versions))
			minorIsNew := true
			minorIsRemoved := true
			for _, ver := range versions {
				_, isNew := newVersionSet[ver]
				_, isRemoved := removedVersionSet[ver]
				if !isNew {
					minorIsNew = false
				}
				if !isRemoved {
					minorIsRemoved = false
				}
				versionUpdates = append(versionUpdates, &model.GoOffsetVersionUpdate{
					Version:   ver,
					IsNew:     isNew,
					IsRemoved: isRemoved,
				})
			}
			if isNewModule {
				minorIsNew = true
			}
			if isRemovedModule {
				minorIsRemoved = true
			}
			minorVersions = append(minorVersions, &model.GoOffsetMinorVersionUpdate{
				MinorVersion: minorKey,
				IsNew:        minorIsNew,
				IsRemoved:    minorIsRemoved,
				Versions:     versionUpdates,
			})
		}

		minVersion, maxVersion := "", ""
		if min != nil {
			minVersion = min.String()
		}
		if max != nil {
			maxVersion = max.String()
		}

		mods = append(mods, &model.GoOffsetModuleUpdate{
			Module:        name,
			IsNew:         isNewModule,
			IsRemoved:     isRemovedModule,
			MinVersion:    minVersion,
			MaxVersion:    maxVersion,
			MinorVersions: minorVersions,
		})
	}

	currentTimestamp := ""
	if !current.Timestamp.IsZero() {
		currentTimestamp = current.Timestamp.UTC().Format(time.RFC3339Nano)
	}
	proposedTimestamp := ""
	if !proposed.Timestamp.IsZero() {
		proposedTimestamp = proposed.Timestamp.UTC().Format(time.RFC3339Nano)
	}

	return &model.GoOffsetsUpdateCheck{
		HasUpdates:        hasUpdates,
		CurrentTimestamp:  currentTimestamp,
		ProposedTimestamp: proposedTimestamp,
		Mods:              mods,
	}
}

func moduleVersionSet(mod *jsonModule) map[string]struct{} {
	byMinor, _, _ := moduleVersions(mod)
	set := make(map[string]struct{})
	for _, versions := range byMinor {
		for _, v := range versions {
			set[v] = struct{}{}
		}
	}
	return set
}

// UpdateGoOffsets writes the provided offsets file content to the
// odigos-go-offsets ConfigMap (same encoding as `odigos pro update-offsets`).
func UpdateGoOffsets(ctx context.Context, content string) error {
	if GetTier(ctx) == model.TierCommunity {
		return fmt.Errorf("custom offsets support is only available in Odigos pro tier")
	}

	return writeGoOffsetsConfigMap(ctx, []byte(content))
}

func writeGoOffsetsConfigMap(ctx context.Context, data []byte) error {
	ns := env.GetCurrentNamespace()

	cm, err := kube.DefaultClient.CoreV1().ConfigMaps(ns).Get(ctx, k8sconsts.GoOffsetsConfigMap, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get ConfigMap %s/%s: %w", ns, k8sconsts.GoOffsetsConfigMap, err)
	}

	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}

	if len(data) == 0 {
		cm.Data[k8sconsts.GoOffsetsFileName] = ""
	} else {
		encoded, err := json.Marshal(string(data))
		if err != nil {
			return fmt.Errorf("failed to encode go offsets: %w", err)
		}
		cm.Data[k8sconsts.GoOffsetsFileName] = string(encoded)
	}

	_, err = kube.DefaultClient.CoreV1().ConfigMaps(ns).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update ConfigMap %s/%s: %w", ns, k8sconsts.GoOffsetsConfigMap, err)
	}

	return nil
}

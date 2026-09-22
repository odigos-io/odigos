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
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetGoOffsets returns parsed go_offset_results.json from the
// odigos-go-offsets ConfigMap in the Odigos namespace.
func GetGoOffsets(ctx context.Context) (*model.GoOffsets, error) {
	content, installed, err := getGoOffsetsRaw(ctx)
	if err != nil {
		return nil, err
	}
	if !installed {
		// Absence is a cluster state the UI can explain, not a read failure: the
		// chart owns this ConfigMap, so a missing one means a partial install or a
		// manual delete. Failing the query here would render as an empty table.
		return &model.GoOffsets{Installed: false, Mods: []*model.GoOffsetModule{}}, nil
	}

	parsed, err := parseGoOffsetsContent(content)
	if err != nil {
		return nil, err
	}

	offsets := goOffsetsToModel(parsed)
	offsets.Installed = true
	return offsets, nil
}

// getGoOffsetsRaw reads the ConfigMap's offsets payload. The second return
// reports whether the ConfigMap exists: callers distinguish "not installed"
// from a genuine read failure, which read as the same empty table otherwise.
func getGoOffsetsRaw(ctx context.Context) (string, bool, error) {
	ns := env.GetCurrentNamespace()

	var cm corev1.ConfigMap
	err := kube.CacheClient.Get(ctx, client.ObjectKey{
		Namespace: ns,
		Name:      k8sconsts.GoOffsetsConfigMap,
	}, &cm)
	if apierrors.IsNotFound(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("failed to get ConfigMap %s/%s: %w", ns, k8sconsts.GoOffsetsConfigMap, err)
	}

	// A ConfigMap without the key is malformed rather than absent, so it stays an
	// error: recreating it would silently discard whatever else is in there.
	content, ok := cm.Data[k8sconsts.GoOffsetsFileName]
	if !ok {
		return "", false, fmt.Errorf("key %q not found in ConfigMap %s/%s", k8sconsts.GoOffsetsFileName, ns, k8sconsts.GoOffsetsConfigMap)
	}

	return content, true, nil
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
	currentRaw, installed, err := getGoOffsetsRaw(ctx)
	if err != nil {
		return nil, err
	}
	// With no ConfigMap the installed set is empty, so every module reads as new
	// and the update becomes the way to put the manifest back.
	current := &versionedModules{Mods: []*jsonModule{}}
	if installed {
		current, err = parseGoOffsetsContent(currentRaw)
		if err != nil {
			return nil, err
		}
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

// Helm refuses to adopt a resource it did not create, so a ConfigMap this
// service recreates has to carry the release's ownership metadata or the next
// `helm upgrade` fails on it. The values are copied from a sibling object the
// chart owns rather than guessed, since the release name is installer's choice.
const (
	helmManagedByLabel        = "app.kubernetes.io/managed-by"
	helmReleaseNameAnnotation = "meta.helm.sh/release-name"
	helmReleaseNsAnnotation   = "meta.helm.sh/release-namespace"
)

func encodeGoOffsets(data []byte) (string, error) {
	if len(data) == 0 {
		return "", nil
	}
	encoded, err := json.Marshal(string(data))
	if err != nil {
		return "", fmt.Errorf("failed to encode go offsets: %w", err)
	}
	return string(encoded), nil
}

func writeGoOffsetsConfigMap(ctx context.Context, data []byte) error {
	ns := env.GetCurrentNamespace()
	configMaps := kube.DefaultClient.CoreV1().ConfigMaps(ns)

	encoded, err := encodeGoOffsets(data)
	if err != nil {
		return err
	}

	cm, err := configMaps.Get(ctx, k8sconsts.GoOffsetsConfigMap, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		// The chart normally owns this object; recreating it here is what makes an
		// update a recovery path after the ConfigMap was deleted.
		return createGoOffsetsConfigMap(ctx, ns, encoded)
	}
	if err != nil {
		return fmt.Errorf("failed to get ConfigMap %s/%s: %w", ns, k8sconsts.GoOffsetsConfigMap, err)
	}

	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}
	cm.Data[k8sconsts.GoOffsetsFileName] = encoded

	updated, err := configMaps.Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update ConfigMap %s/%s: %w", ns, k8sconsts.GoOffsetsConfigMap, err)
	}

	awaitGoOffsetsCache(ctx, ns, updated.ResourceVersion)
	return nil
}

// Writes go through the live API client while reads are served from an informer
// cache, so a write that the API server has accepted is not necessarily visible
// to the next read. The UI refetches the moment this mutation resolves, which
// without this wait hands it back the state it just replaced. Best effort: the
// write already succeeded, so a cache that never catches up is not an error.
func awaitGoOffsetsCache(ctx context.Context, ns, resourceVersion string) {
	const (
		timeout  = 3 * time.Second
		interval = 25 * time.Millisecond
	)

	deadline := time.Now().Add(timeout)
	for {
		var cached corev1.ConfigMap
		err := kube.CacheClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: k8sconsts.GoOffsetsConfigMap}, &cached)
		if err == nil && cached.ResourceVersion == resourceVersion {
			return
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			return
		}
		time.Sleep(interval)
	}
}

func createGoOffsetsConfigMap(ctx context.Context, ns, encoded string) error {
	labels := map[string]string{k8sconsts.OdigosSystemLabelKey: "true"}
	annotations := map[string]string{}

	// Best effort: without the sibling the ConfigMap is still created and odiglet
	// can mount it, but a later `helm upgrade` needs the release to adopt it.
	if owner, err := kube.DefaultClient.CoreV1().ConfigMaps(ns).Get(ctx, consts.OdigosConfigurationName, metav1.GetOptions{}); err == nil {
		if v := owner.Labels[helmManagedByLabel]; v != "" {
			labels[helmManagedByLabel] = v
		}
		for _, key := range []string{helmReleaseNameAnnotation, helmReleaseNsAnnotation} {
			if v := owner.Annotations[key]; v != "" {
				annotations[key] = v
			}
		}
	}

	created, err := kube.DefaultClient.CoreV1().ConfigMaps(ns).Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        k8sconsts.GoOffsetsConfigMap,
			Namespace:   ns,
			Labels:      labels,
			Annotations: annotations,
		},
		Data: map[string]string{k8sconsts.GoOffsetsFileName: encoded},
	}, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create ConfigMap %s/%s: %w", ns, k8sconsts.GoOffsetsConfigMap, err)
	}

	awaitGoOffsetsCache(ctx, ns, created.ResourceVersion)
	return nil
}

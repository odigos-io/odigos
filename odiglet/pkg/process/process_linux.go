package process

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"

	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
	corev1 "k8s.io/api/core/v1"
)

func groupByCgroup(pcs []PodContainer) (map[PodContainerKey]map[int]struct{}, error) {
	result := make(map[PodContainerKey]map[int]struct{}, len(pcs))
	for _, pc := range pcs {
		if pc.ContainerID == "" {
			continue
		}
		pids, err := pidsInContainerByCgroup(hostCgroupLayout, pc)
		if err != nil {
			if errors.Is(err, ErrCgroupMissing) {
				continue
			}
			return nil, fmt.Errorf("pids in container %s/%s: %w", pc.PodUID, pc.ContainerName, err)
		}
		if len(pids) == 0 {
			continue
		}
		set := make(map[int]struct{}, len(pids))
		for _, pid := range pids {
			set[pid] = struct{}{}
		}
		result[pc.PodContainerKey] = set
	}
	return result, nil
}

var groupByCgroupFunc = groupByCgroup

// groupByProcMountInfo is the legacy fallback: a single /proc scan
// that checks each PID's mountinfo for the k8s service-account mount
// path identifying its pod+container. Slower per-PID but only walks
// /proc once for any size of pcs.
func groupByProcMountInfo(pcs []PodContainer) (map[PodContainerKey]map[int]struct{}, error) {
	keys := make([]PodContainerKey, 0, len(pcs))
	for _, pc := range pcs {
		keys = append(keys, pc.PodContainerKey)
	}
	groups, err := process.Group(isInPodContainersBatchPredicate(keys))
	if err != nil {
		return nil, fmt.Errorf("failed to group processes by (pod, container) :%w", err)
	}
	return groups, nil
}

var groupByProcMountInfoFunc = groupByProcMountInfo

func isInPodContainersBatchPredicate(keys []PodContainerKey) func(int) (PodContainerKey, bool) {
	expectedMountByPodContainer := make(map[PodContainerKey][]byte, len(keys))
	for _, k := range keys {
		expectedMount := fmt.Sprintf("%s/containers/%s/", k.PodUID, k.ContainerName)
		expectedMountByPodContainer[k] = []byte(expectedMount)
	}

	return func(pid int) (PodContainerKey, bool) {
		mountInfoFile := process.ProcFilePath(pid, "mountinfo")
		f, err := os.Open(mountInfoFile)
		if err != nil {
			return PodContainerKey{}, false
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			for pc, mountPath := range expectedMountByPodContainer {
				if bytes.Contains(scanner.Bytes(), mountPath) {
					return pc, true
				}
			}
		}
		return PodContainerKey{}, false
	}
}

type PodContainerKey struct {
	PodUID        string
	ContainerName string
}

type PodContainer struct {
	PodContainerKey
	QOSClass    corev1.PodQOSClass
	ContainerID string
}

// GroupByPodContainer groups all the current active processes by
// (podUID, containerName) using the provided list of PodContainers to
// filter relevant processes. Processes that do not belong to any of
// the provided PodContainers are ignored.
func GroupByPodContainer(pcs []PodContainer) (map[PodContainerKey]map[int]struct{}, error) {
	// if we have the cgroup layout available,
	// use it for grouping as it's more efficient.
	if hostCgroupLayout.Valid {
		groups, err := groupByCgroupFunc(pcs)
		// if we had an error or no groups, fallback to the legacy proc mountinfo parsing.
		// currently we fallback even when err is nil and no matches were found,
		// to avoid possible regressions due to the cgroup path resolution logic.
		switch {
		case err == nil && len(groups) > 0:
			missingPCs := podContainersMissingGroups(pcs, groups)
			if len(missingPCs) > 0 {
				fallbackGroups, fallbackErr := groupByProcMountInfoFunc(missingPCs)
				if fallbackErr != nil {
					commonlogger.LoggerCompat().Warn("failed to resolve missing process groups from proc scan, using partial cgroup result", "error", fallbackErr, "missing-pod-containers", missingPCs)
				} else {
					mergeProcessGroups(groups, fallbackGroups)
				}
			}
			commonlogger.LoggerCompat().Debug("found process groups based on cgroups", "groups", len(groups), "pod-containers", pcs)
			return groups, nil
		case err != nil:
			commonlogger.LoggerCompat().Warn("failed to perfrom pid resolution based on cgroup, fallback to proc scan", "error", err)
		}
	}
	// fallback to /proc/<pid>/mountinfo parsing
	return groupByProcMountInfoFunc(pcs)
}

func podContainersMissingGroups(pcs []PodContainer, groups map[PodContainerKey]map[int]struct{}) []PodContainer {
	missing := make([]PodContainer, 0)
	seenMissing := make(map[PodContainerKey]struct{}, len(pcs))
	for _, pc := range pcs {
		if _, ok := groups[pc.PodContainerKey]; ok {
			continue
		}
		if _, seen := seenMissing[pc.PodContainerKey]; seen {
			continue
		}
		seenMissing[pc.PodContainerKey] = struct{}{}
		missing = append(missing, pc)
	}
	return missing
}

func mergeProcessGroups(dst, src map[PodContainerKey]map[int]struct{}) {
	for key, pids := range src {
		if len(pids) == 0 {
			continue
		}
		dstPIDs, ok := dst[key]
		if !ok {
			dstPIDs = make(map[int]struct{}, len(pids))
			dst[key] = dstPIDs
		}
		for pid := range pids {
			dstPIDs[pid] = struct{}{}
		}
	}
}

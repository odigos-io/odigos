package status

import (
	"testing"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/computed"
	"github.com/odigos-io/odigos/frontend/graph/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	agentHashChangedAt = metav1.NewTime(time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC))
	podCreatedBefore   = metav1.NewTime(agentHashChangedAt.Add(-time.Hour))
	podCreatedAfter    = metav1.NewTime(agentHashChangedAt.Add(time.Hour))
)

// agentHashLabel is nil when the pod carries no odigos label at all, which is
// how "the agent was never injected into this pod" is represented.
func injectionPod(agentHashLabel *string, creationTime metav1.Time) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "checkout-abc123",
			Namespace:         "default",
			CreationTimestamp: creationTime,
		},
	}
	if agentHashLabel != nil {
		pod.Labels = map[string]string{k8sconsts.OdigosAgentsMetaHashLabel: *agentHashLabel}
	}
	return pod
}

type injectionICOptions struct {
	name                  string
	agentInjectionEnabled bool
	agentsMetaHash        string
	injectionOptional     bool
	hashChangedTime       *metav1.Time
}

func injectionIC(opts injectionICOptions) *v1alpha1.InstrumentationConfig {
	name := opts.name
	if name == "" {
		name = "deployment-checkout"
	}
	return &v1alpha1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec: v1alpha1.InstrumentationConfigSpec{
			AgentInjectionEnabled:        opts.agentInjectionEnabled,
			AgentsMetaHash:               opts.agentsMetaHash,
			PodManifestInjectionOptional: opts.injectionOptional,
			AgentsMetaHashChangedTime:    opts.hashChangedTime,
		},
	}
}

// This decision tree is what the UI turns into "everything is fine" versus
// "rollout this workload". Each outcome differs from its neighbours only in the
// severity and the wording of the action to take, so the exact message is part
// of the contract: two outcomes share a reason and are distinguishable only by
// their message.
func TestCalculatePodAgentInjectedStatus(t *testing.T) {
	matchingHash := "hash-current"
	staleHash := "hash-old"
	emptyHash := ""

	tests := []struct {
		name                    string
		pod                     *corev1.Pod
		ic                      *v1alpha1.InstrumentationConfig
		automaticRolloutEnabled bool
		wantAgentInjected       bool
		wantStatus              model.DesiredStateProgress
		wantReason              PodAgentInjectedReason
		wantMessage             string
	}{
		{
			name:              "not marked for instrumentation, no agent",
			pod:               injectionPod(nil, podCreatedAfter),
			ic:                nil,
			wantAgentInjected: false,
			wantStatus:        model.DesiredStateProgressSuccess,
			wantReason:        PodAgentInjectedReasonNotMarkedNotInjected,
			wantMessage:       "source is not marked for instrumentation; odigos agent is not injected to pod (expected)",
		},
		{
			name:                    "not marked for instrumentation, agent injected, automatic rollout",
			pod:                     injectionPod(&matchingHash, podCreatedAfter),
			ic:                      nil,
			automaticRolloutEnabled: true,
			wantAgentInjected:       true,
			wantStatus:              model.DesiredStateProgressWaiting,
			wantReason:              PodAgentInjectedReasonNotMarkedAutoRollout,
			wantMessage:             "source is not marked for instrumentation and odigos agent is injected; this source will be rolled out automatically by odigos to replace with new uninstrumented pods",
		},
		{
			name:              "not marked for instrumentation, agent injected, manual rollout",
			pod:               injectionPod(&matchingHash, podCreatedAfter),
			ic:                nil,
			wantAgentInjected: true,
			wantStatus:        model.DesiredStateProgressNotice,
			wantReason:        PodAgentInjectedReasonNotMarkedManualRollout,
			wantMessage:       "source is not marked for instrumentation and odigos agent is injected; rollout this source to start new uninstrumented pods",
		},
		{
			name:              "agent injection disabled, no agent",
			pod:               injectionPod(nil, podCreatedAfter),
			ic:                injectionIC(injectionICOptions{agentInjectionEnabled: false}),
			wantAgentInjected: false,
			wantStatus:        model.DesiredStateProgressSuccess,
			wantReason:        PodAgentInjectedReasonDisabledNotInjected,
			wantMessage:       "source is disabled for agent injection; odigos agent is not injected (expected)",
		},
		{
			name:                    "agent injection disabled, agent injected, automatic rollout",
			pod:                     injectionPod(&matchingHash, podCreatedAfter),
			ic:                      injectionIC(injectionICOptions{agentInjectionEnabled: false}),
			automaticRolloutEnabled: true,
			wantAgentInjected:       true,
			wantStatus:              model.DesiredStateProgressWaiting,
			wantReason:              PodAgentInjectedReasonDisabledAutoRollout,
			wantMessage:             "Deployment is disabled for agent injection but odigos agent is injected; this Deployment will be rolled out automatically by odigos",
		},
		{
			name:              "agent injection disabled, agent injected, manual rollout",
			pod:               injectionPod(&matchingHash, podCreatedAfter),
			ic:                injectionIC(injectionICOptions{agentInjectionEnabled: false, name: "statefulset-cart"}),
			wantAgentInjected: true,
			wantStatus:        model.DesiredStateProgressNotice,
			wantReason:        PodAgentInjectedReasonDisabledManualRollout,
			wantMessage:       "StatefulSet is disabled for agent injection but odigos agent is injected; rollout this StatefulSet to replace with new uninstrumented pods",
		},
		{
			name:              "agent injected with the current hash",
			pod:               injectionPod(&matchingHash, podCreatedAfter),
			ic:                injectionIC(injectionICOptions{agentInjectionEnabled: true, agentsMetaHash: matchingHash}),
			wantAgentInjected: true,
			wantStatus:        model.DesiredStateProgressSuccess,
			wantReason:        PodAgentInjectedReasonSuccessfullyInjected,
			wantMessage:       "odigos agent is successfully injected to this pod",
		},
		{
			// A hash has not been computed for the workload yet and the pod
			// carries the label with an empty value, which counts as up to date.
			name:              "agent label present and no hash computed yet",
			pod:               injectionPod(&emptyHash, podCreatedAfter),
			ic:                injectionIC(injectionICOptions{agentInjectionEnabled: true}),
			wantAgentInjected: true,
			wantStatus:        model.DesiredStateProgressSuccess,
			wantReason:        PodAgentInjectedReasonSuccessfullyInjected,
			wantMessage:       "odigos agent is successfully injected to this pod",
		},
		{
			// A "no restart required" agent is applied to running pods, so a
			// stale hash in the manifest is not a problem to report.
			name:              "stale hash with optional pod manifest injection",
			pod:               injectionPod(&staleHash, podCreatedAfter),
			ic:                injectionIC(injectionICOptions{agentInjectionEnabled: true, agentsMetaHash: matchingHash, injectionOptional: true}),
			wantAgentInjected: true,
			wantStatus:        model.DesiredStateProgressSuccess,
			wantReason:        PodAgentInjectedReasonPodManifestInjectionOptional,
			wantMessage:       "this agent is automatically enabled in running pod",
		},
		{
			name:                    "stale hash, automatic rollout",
			pod:                     injectionPod(&staleHash, podCreatedAfter),
			ic:                      injectionIC(injectionICOptions{agentInjectionEnabled: true, agentsMetaHash: matchingHash}),
			automaticRolloutEnabled: true,
			wantAgentInjected:       true,
			wantStatus:              model.DesiredStateProgressWaiting,
			wantReason:              PodAgentInjectedReasonOutOfDateAutoRollout,
			wantMessage:             "odigos agent is not up to date; this Deployment will be rolled out automatically by odigos",
		},
		{
			name:              "stale hash, manual rollout",
			pod:               injectionPod(&staleHash, podCreatedAfter),
			ic:                injectionIC(injectionICOptions{agentInjectionEnabled: true, agentsMetaHash: matchingHash}),
			wantAgentInjected: true,
			wantStatus:        model.DesiredStateProgressNotice,
			wantReason:        PodAgentInjectedReasonOutOfDateManualRollout,
			wantMessage:       "odigos agent is not up to date; rollout this Deployment to start new pods with latest agent version",
		},
		{
			name:              "no agent label but injection is optional",
			pod:               injectionPod(nil, podCreatedAfter),
			ic:                injectionIC(injectionICOptions{agentInjectionEnabled: true, agentsMetaHash: matchingHash, injectionOptional: true}),
			wantAgentInjected: false,
			wantStatus:        model.DesiredStateProgressSuccess,
			wantReason:        PodAgentInjectedReasonPodManifestInjectionOptional,
			wantMessage:       "this agent is automatically enabled in running pod",
		},
		{
			// Sources instrumented before the hash-changed timestamp existed.
			name:              "no agent label and no hash changed time",
			pod:               injectionPod(nil, podCreatedAfter),
			ic:                injectionIC(injectionICOptions{agentInjectionEnabled: true, agentsMetaHash: matchingHash}),
			wantAgentInjected: false,
			wantStatus:        model.DesiredStateProgressNotice,
			wantReason:        PodAgentInjectedReasonEnabledNotInjected,
			wantMessage:       "source is enabled for agent injection but odigos agent was not injected; rollout the workload to replace this pod with an instrumented one",
		},
		{
			name: "pod older than the agent, automatic rollout",
			pod:  injectionPod(nil, podCreatedBefore),
			ic: injectionIC(injectionICOptions{
				agentInjectionEnabled: true, agentsMetaHash: matchingHash, hashChangedTime: &agentHashChangedAt,
			}),
			automaticRolloutEnabled: true,
			wantAgentInjected:       false,
			wantStatus:              model.DesiredStateProgressWaiting,
			wantReason:              PodAgentInjectedReasonEnabledAfterPodCreatedAutoRollout,
			wantMessage:             "old pod - created before agent was enabled; will be rolled out automatically by odigos",
		},
		{
			name: "pod older than the agent, manual rollout",
			pod:  injectionPod(nil, podCreatedBefore),
			ic: injectionIC(injectionICOptions{
				agentInjectionEnabled: true, agentsMetaHash: matchingHash, hashChangedTime: &agentHashChangedAt,
			}),
			wantAgentInjected: false,
			wantStatus:        model.DesiredStateProgressNotice,
			wantReason:        PodAgentInjectedReasonEnabledAfterPodCreatedManualRollout,
			wantMessage:       "old pod - created before agent was enabled; rollout the workload to replace this pod with an instrumented one",
		},
		{
			// Same reason as "no hash changed time" above but a different
			// message: this pod started after the agent was enabled, so the
			// webhook should have injected it and something went wrong.
			name: "pod newer than the agent but not injected",
			pod:  injectionPod(nil, podCreatedAfter),
			ic: injectionIC(injectionICOptions{
				agentInjectionEnabled: true, agentsMetaHash: matchingHash, hashChangedTime: &agentHashChangedAt,
			}),
			wantAgentInjected: false,
			wantStatus:        model.DesiredStateProgressNotice,
			wantReason:        PodAgentInjectedReasonEnabledNotInjected,
			wantMessage:       "agent is not injected to this pod; check for instrumentor component health and rollout the workload to replace this pod with an instrumented one",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agentInjected, got := CalculatePodAgentInjectedStatus(tt.pod, tt.ic, tt.automaticRolloutEnabled)

			// The returned flag drives the pod level odigos health, which uses
			// it to tell "agent missing" from "agent present but unwanted".
			if agentInjected != tt.wantAgentInjected {
				t.Fatalf("expected agentInjected %t, got %t", tt.wantAgentInjected, agentInjected)
			}
			if got == nil {
				t.Fatal("expected a status, got nil")
			}
			if got.Name != PodAgentInjectionStatus {
				t.Fatalf("expected name %q, got %q", PodAgentInjectionStatus, got.Name)
			}
			if got.Status != tt.wantStatus {
				t.Fatalf("expected status %q, got %q", tt.wantStatus, got.Status)
			}
			if reasonOf(t, got) != string(tt.wantReason) {
				t.Fatalf("expected reason %q, got %q", tt.wantReason, reasonOf(t, got))
			}
			if got.Message != tt.wantMessage {
				t.Fatalf("expected message:\n%q\ngot:\n%q", tt.wantMessage, got.Message)
			}
		})
	}
}

// A pod created at exactly the hash change timestamp is treated as new enough,
// so it is not reported as an old pod awaiting a rollout.
func TestCalculatePodAgentInjectedStatusPodCreatedAtTheHashChangeTime(t *testing.T) {
	ic := injectionIC(injectionICOptions{
		agentInjectionEnabled: true, agentsMetaHash: "hash-current", hashChangedTime: &agentHashChangedAt,
	})

	_, got := CalculatePodAgentInjectedStatus(injectionPod(nil, agentHashChangedAt), ic, false)

	if reasonOf(t, got) != string(PodAgentInjectedReasonEnabledNotInjected) {
		t.Fatalf("expected reason %q, got %q", PodAgentInjectedReasonEnabledNotInjected, reasonOf(t, got))
	}
}

func cachedPodWithInjectionReason(podName string, reason PodAgentInjectedReason) computed.CachedPod {
	reasonStr := string(reason)
	return computed.CachedPod{
		PodName: podName,
		AgentInjectedStatus: &model.DesiredConditionStatus{
			Name:       PodAgentInjectionStatus,
			ReasonEnum: &reasonStr,
		},
	}
}

// The workload level condition is picked by walking a fixed priority list of
// per-pod reasons, not by severity. One pod in the wrong state has to be able
// to raise the whole workload's condition, otherwise a partially broken
// rollout looks completely healthy.
func TestCalculateAgentInjectedStatusSingleReason(t *testing.T) {
	tests := []struct {
		reason      PodAgentInjectedReason
		wantStatus  model.DesiredStateProgress
		wantReason  AgentInjectionReason
		wantMessage string
	}{
		{
			reason:      PodAgentInjectedReasonNotMarkedManualRollout,
			wantStatus:  model.DesiredStateProgressNotice,
			wantReason:  AgentInjectionReasonSomePodsAgentInjectedRolloutNeeded,
			wantMessage: "source not marked for instrumentation but 1/2 pods are running odigos agent; rollout this source to replace these pods with uninstrumented ones",
		},
		{
			reason:      PodAgentInjectedReasonNotMarkedAutoRollout,
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  AgentInjectionReasonSomePodsAgentInjectedWaitingForAutoRollout,
			wantMessage: "odigos agent is injected in 1/2 pods; odigos will roll out these pods automatically",
		},
		{
			reason:      PodAgentInjectedReasonNotMarkedNotInjected,
			wantStatus:  model.DesiredStateProgressSuccess,
			wantReason:  AgentInjecteonReasonNotInjectedAsExpected,
			wantMessage: "odigos agent is not injected as expected since source is not marked for instrumentation",
		},
		{
			reason:      PodAgentInjectedReasonDisabledAutoRollout,
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  AgentInjectionReasonSomePodsAgentInjectedWaitingForAutoRollout,
			wantMessage: "source is disabled for agent injection but 1/2 pods are running odigos agent; odigos will roll out these pods automatically",
		},
		{
			reason:      PodAgentInjectedReasonDisabledManualRollout,
			wantStatus:  model.DesiredStateProgressNotice,
			wantReason:  AgentInjectionReasonSomePodsAgentInjectedRolloutNeeded,
			wantMessage: "source is disabled for agent injection but 1/2 pods are running odigos agent; rollout this source to replace these pods with uninstrumented ones",
		},
		{
			reason:      PodAgentInjectedReasonDisabledNotInjected,
			wantStatus:  model.DesiredStateProgressSuccess,
			wantReason:  AgentInjecteonReasonNotInjectedAsExpected,
			wantMessage: "source is disabled for agent injection but 1/2 pods are running odigos agent; rollout this source to replace these pods with uninstrumented ones",
		},
		{
			reason:      PodAgentInjectedReasonPodManifestInjectionOptional,
			wantStatus:  model.DesiredStateProgressSuccess,
			wantReason:  AgentInjectionReason(PodAgentInjectedReasonPodManifestInjectionOptional),
			wantMessage: "this agent is automatically enabled in running pod",
		},
		{
			reason:      PodAgentInjectedReasonEnabledNotInjected,
			wantStatus:  model.DesiredStateProgressNotice,
			wantReason:  AgentInjectionReasonSomePodsAgentNotInjectedRolloutNeeded,
			wantMessage: "1/2 pods are running without odigos agent and require restart to apply instrumentation; check instrumentor component health and trigger a rollout",
		},
		{
			reason:      PodAgentInjectedReasonEnabledAfterPodCreatedManualRollout,
			wantStatus:  model.DesiredStateProgressNotice,
			wantReason:  AgentInjectionReasonSomePodsAgentNotInjectedRolloutNeeded,
			wantMessage: "1/2 pods are running without odigos agent and require restart to apply instrumentation; trigger a rollout to replace these pods with instrumented ones",
		},
		{
			reason:      PodAgentInjectedReasonEnabledAfterPodCreatedAutoRollout,
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  AgentInjectionReasonSomePodsAgentNotInjectedRolloutNeeded,
			wantMessage: "1/2 pods are running without odigos agent and require restart to apply instrumentation; trigger a rollout to replace these pods with instrumented ones",
		},
		{
			reason:      PodAgentInjectedReasonOutOfDateManualRollout,
			wantStatus:  model.DesiredStateProgressNotice,
			wantReason:  AgentInjectionReasonSomePodsAgentOutOfDateRolloutNeeded,
			wantMessage: "1/2 pods are running without odigos agent and require restart to apply instrumentation; trigger a rollout to replace these pods with instrumented ones",
		},
		{
			reason:      PodAgentInjectedReasonOutOfDateAutoRollout,
			wantStatus:  model.DesiredStateProgressWaiting,
			wantReason:  AgentInjectionReasonSomePodsAgentOutOfDateWaitingForAutoRollout,
			wantMessage: "1/2 pods are running without odigos agent and require restart to apply instrumentation; odigos will roll out these pods automatically",
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.reason), func(t *testing.T) {
			// The second pod is successfully injected, so the counts in the
			// message must be "1/2" rather than "2/2" or "1/1".
			pods := []computed.CachedPod{
				cachedPodWithInjectionReason("pod-1", tt.reason),
				cachedPodWithInjectionReason("pod-2", PodAgentInjectedReasonSuccessfullyInjected),
			}

			got := CalculateAgentInjectedStatus(&v1alpha1.InstrumentationConfig{}, pods)
			if got == nil {
				t.Fatal("expected a status, got nil")
			}
			if got.Name != AgentInjectedStatus {
				t.Fatalf("expected name %q, got %q", AgentInjectedStatus, got.Name)
			}
			if got.Status != tt.wantStatus {
				t.Fatalf("expected status %q, got %q", tt.wantStatus, got.Status)
			}
			if reasonOf(t, got) != string(tt.wantReason) {
				t.Fatalf("expected reason %q, got %q", tt.wantReason, reasonOf(t, got))
			}
			if got.Message != tt.wantMessage {
				t.Fatalf("expected message:\n%q\ngot:\n%q", tt.wantMessage, got.Message)
			}
		})
	}
}

// Each adjacent pair in the priority list is checked with both reasons present,
// so a reordering of the chain fails here even though every individual reason
// still maps to the same condition.
func TestCalculateAgentInjectedStatusReasonPriority(t *testing.T) {
	priority := []PodAgentInjectedReason{
		PodAgentInjectedReasonNotMarkedManualRollout,
		PodAgentInjectedReasonNotMarkedAutoRollout,
		PodAgentInjectedReasonNotMarkedNotInjected,
		PodAgentInjectedReasonDisabledAutoRollout,
		PodAgentInjectedReasonDisabledManualRollout,
		PodAgentInjectedReasonDisabledNotInjected,
		PodAgentInjectedReasonPodManifestInjectionOptional,
		PodAgentInjectedReasonEnabledNotInjected,
		PodAgentInjectedReasonEnabledAfterPodCreatedManualRollout,
		PodAgentInjectedReasonEnabledAfterPodCreatedAutoRollout,
		PodAgentInjectedReasonOutOfDateManualRollout,
		PodAgentInjectedReasonOutOfDateAutoRollout,
	}

	// The condition each reason produces on its own, used as the expectation
	// for the winner of every pair.
	expectedFor := func(reason PodAgentInjectedReason) *model.DesiredConditionStatus {
		return CalculateAgentInjectedStatus(&v1alpha1.InstrumentationConfig{}, []computed.CachedPod{
			cachedPodWithInjectionReason("pod-1", reason),
			cachedPodWithInjectionReason("pod-2", PodAgentInjectedReasonSuccessfullyInjected),
		})
	}

	for i := 1; i < len(priority); i++ {
		higher, lower := priority[i-1], priority[i]
		t.Run(string(higher)+" over "+string(lower), func(t *testing.T) {
			want := expectedFor(higher)

			got := CalculateAgentInjectedStatus(&v1alpha1.InstrumentationConfig{}, []computed.CachedPod{
				// the lower priority reason comes first in the pod list, so the
				// result cannot come from the pod ordering.
				cachedPodWithInjectionReason("pod-1", lower),
				cachedPodWithInjectionReason("pod-2", higher),
			})

			if got == nil {
				t.Fatal("expected a status, got nil")
			}
			if reasonOf(t, got) != reasonOf(t, want) || got.Status != want.Status {
				t.Fatalf("expected %q/%q from %q, got %q/%q",
					want.Status, reasonOf(t, want), higher, got.Status, reasonOf(t, got))
			}
		})
	}
}

func TestCalculateAgentInjectedStatusNoPods(t *testing.T) {
	got := CalculateAgentInjectedStatus(&v1alpha1.InstrumentationConfig{}, nil)
	if got == nil {
		t.Fatal("expected a status, got nil")
	}
	if got.Status != model.DesiredStateProgressIrrelevant {
		t.Fatalf("expected status %q, got %q", model.DesiredStateProgressIrrelevant, got.Status)
	}
	if reasonOf(t, got) != string(AgentInjectionReasonNoRunningPods) {
		t.Fatalf("expected reason %q, got %q", AgentInjectionReasonNoRunningPods, reasonOf(t, got))
	}
	if got.Message != "no pods found for this source" {
		t.Fatalf("unexpected message %q", got.Message)
	}
}

func TestCalculateAgentInjectedStatusAllPodsInjected(t *testing.T) {
	pods := []computed.CachedPod{
		cachedPodWithInjectionReason("pod-1", PodAgentInjectedReasonSuccessfullyInjected),
		cachedPodWithInjectionReason("pod-2", PodAgentInjectedReasonSuccessfullyInjected),
		cachedPodWithInjectionReason("pod-3", PodAgentInjectedReasonSuccessfullyInjected),
	}

	got := CalculateAgentInjectedStatus(&v1alpha1.InstrumentationConfig{}, pods)
	if got.Status != model.DesiredStateProgressSuccess {
		t.Fatalf("expected status %q, got %q", model.DesiredStateProgressSuccess, got.Status)
	}
	if reasonOf(t, got) != string(PodAgentInjectedReasonSuccessfullyInjected) {
		t.Fatalf("expected reason %q, got %q", PodAgentInjectedReasonSuccessfullyInjected, reasonOf(t, got))
	}
	if got.Message != "all 3 pods are instrumented as expected" {
		t.Fatalf("unexpected message %q", got.Message)
	}
}

// Pods whose injection status was not computed are ignored rather than
// counted, so they cannot invent a reason for the workload.
func TestCalculateAgentInjectedStatusPodsWithoutAReason(t *testing.T) {
	tests := []struct {
		name string
		pods []computed.CachedPod
	}{
		{
			name: "no injection status",
			pods: []computed.CachedPod{{PodName: "pod-1"}, {PodName: "pod-2"}},
		},
		{
			name: "injection status without a reason",
			pods: []computed.CachedPod{
				{PodName: "pod-1", AgentInjectedStatus: &model.DesiredConditionStatus{Name: PodAgentInjectionStatus}},
				{PodName: "pod-2", AgentInjectedStatus: &model.DesiredConditionStatus{Name: PodAgentInjectionStatus}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateAgentInjectedStatus(&v1alpha1.InstrumentationConfig{}, tt.pods)
			if got.Status != model.DesiredStateProgressSuccess {
				t.Fatalf("expected status %q, got %q", model.DesiredStateProgressSuccess, got.Status)
			}
			// the pods are still counted in the total.
			if got.Message != "all 2 pods are instrumented as expected" {
				t.Fatalf("unexpected message %q", got.Message)
			}
		})
	}
}

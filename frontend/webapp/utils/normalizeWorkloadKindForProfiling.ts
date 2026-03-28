/**
 * Mirrors backend `normalizeWorkloadKind` in `frontend/services/collector_profiles/handlers.go`
 * so REST paths use the same `namespace/Kind/name` key as OTLP `SourceKeyFromResource`.
 */
export function normalizeWorkloadKindForProfiling(kindStr: string): string {
  const k = kindStr.trim().toLowerCase();
  switch (k) {
    case 'deployment':
      return 'Deployment';
    case 'daemonset':
      return 'DaemonSet';
    case 'statefulset':
      return 'StatefulSet';
    case 'cronjob':
      return 'CronJob';
    case 'job':
      return 'Job';
    case 'deploymentconfig':
      return 'DeploymentConfig';
    case 'rollout':
      return 'Rollout';
    case 'namespace':
      return 'Namespace';
    case 'staticpod':
      return 'StaticPod';
    default:
      return kindStr.trim();
  }
}

import { K8sActualSource } from '@/types';

// while odigos lists language per container, we want to aggregate one single language for the workload.
// the process is mostly heuristic, we iterate over the containers and return the first valid language we find.
// there are additional cases for when the workload programming language is not available.
export enum WORKLOAD_PROGRAMMING_LANGUAGES {
  JAVA = 'java',
  GO = 'go',
  JAVASCRIPT = 'javascript',
  PYTHON = 'python',
  DOTNET = 'dotnet',
  MYSQL = 'mysql',
  NGINX = 'nginx',
  IGNORED = 'ignored',
  UNKNOWN = 'unknown', // language detection completed but could not find a supported language
  PROCESSING = 'processing', // language detection is not yet complotted, data is not available
  NO_CONTAINERS = 'no containers', // language detection completed but no containers found or they are ignored
  NO_RUNNING_PODS = 'no running pods', // no running pods are available for language detection
}

export const getMainContainerLanguage = (source: K8sActualSource): WORKLOAD_PROGRAMMING_LANGUAGES => {
  const ia = source?.instrumentedApplicationDetails;

  if (!ia) {
    if (source?.numberOfInstances > 0) {
      return WORKLOAD_PROGRAMMING_LANGUAGES.PROCESSING;
    } else {
      return WORKLOAD_PROGRAMMING_LANGUAGES.NO_RUNNING_PODS;
    }
  }

  const containers = ia.containers;
  if (!containers) {
    return WORKLOAD_PROGRAMMING_LANGUAGES.PROCESSING;
  }

  // we will filter out the ignored languages as we don't want to account them in the main language
  const noneIgnoredLanguages = containers.filter((container) => container.language !== 'ignored');
  if (noneIgnoredLanguages.length === 0) {
    return WORKLOAD_PROGRAMMING_LANGUAGES.NO_CONTAINERS;
  }

  // find the first container with valid language
  const mainContainer = noneIgnoredLanguages.find((container) => container.language !== 'unknown');
  if (!mainContainer) {
    return WORKLOAD_PROGRAMMING_LANGUAGES.UNKNOWN; // no valid language found, return the first one
  }
  return mainContainer.language as WORKLOAD_PROGRAMMING_LANGUAGES;
};

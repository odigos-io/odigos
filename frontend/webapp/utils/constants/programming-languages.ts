import { K8sActualSource } from '@/types';

const BASE_URL = 'https://d1n7d4xz7fr8b4.cloudfront.net/';

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
  UNKNOWN = 'unknown', // language detection completed but could not find a supported language
  PROCESSING = 'processing', // language detection is not yet complotted, data is not available
  NO_CONTAINERS = 'no containers', // language detection completed but no containers found or they are ignored
  NO_RUNNING_PODS = 'no running pods', // no running pods are available for language detection
  NGINX = 'nginx',
}

export const LANGUAGES_LOGOS: Record<WORKLOAD_PROGRAMMING_LANGUAGES, string> = {
  [WORKLOAD_PROGRAMMING_LANGUAGES.JAVA]: `${BASE_URL}java.svg`,
  [WORKLOAD_PROGRAMMING_LANGUAGES.GO]: `${BASE_URL}go.svg`,
  [WORKLOAD_PROGRAMMING_LANGUAGES.JAVASCRIPT]: `${BASE_URL}nodejs.svg`,
  [WORKLOAD_PROGRAMMING_LANGUAGES.PYTHON]: `${BASE_URL}python.svg`,
  [WORKLOAD_PROGRAMMING_LANGUAGES.DOTNET]: `${BASE_URL}dotnet.svg`,
  [WORKLOAD_PROGRAMMING_LANGUAGES.MYSQL]: `${BASE_URL}mysql.svg`,
  [WORKLOAD_PROGRAMMING_LANGUAGES.UNKNOWN]: `${BASE_URL}default.svg`, // TODO: good icon
  [WORKLOAD_PROGRAMMING_LANGUAGES.PROCESSING]: `${BASE_URL}default.svg`, // TODO: good icon
  [WORKLOAD_PROGRAMMING_LANGUAGES.NO_CONTAINERS]: `${BASE_URL}default.svg`, // TODO: good icon
  [WORKLOAD_PROGRAMMING_LANGUAGES.NO_RUNNING_PODS]: `${BASE_URL}default.svg`, // TODO: good icon
  [WORKLOAD_PROGRAMMING_LANGUAGES.NGINX]: `${BASE_URL}nginx.svg`,
};

export const LANGUAGES_COLORS: Record<WORKLOAD_PROGRAMMING_LANGUAGES, string> =
  {
    [WORKLOAD_PROGRAMMING_LANGUAGES.JAVA]: '#B07219',
    [WORKLOAD_PROGRAMMING_LANGUAGES.GO]: '#00ADD8',
    [WORKLOAD_PROGRAMMING_LANGUAGES.JAVASCRIPT]: '#F7DF1E',
    [WORKLOAD_PROGRAMMING_LANGUAGES.PYTHON]: '#306998',
    [WORKLOAD_PROGRAMMING_LANGUAGES.DOTNET]: '#512BD4',
    [WORKLOAD_PROGRAMMING_LANGUAGES.MYSQL]: '#00758F',
    [WORKLOAD_PROGRAMMING_LANGUAGES.UNKNOWN]: '#8b92a6',
    [WORKLOAD_PROGRAMMING_LANGUAGES.PROCESSING]: '#3367d9',
    [WORKLOAD_PROGRAMMING_LANGUAGES.NO_CONTAINERS]: '#111111',
    [WORKLOAD_PROGRAMMING_LANGUAGES.NO_RUNNING_PODS]: '#666666',
    [WORKLOAD_PROGRAMMING_LANGUAGES.NGINX]: '#009237',
  };

export const getMainContainerLanguage = (
  source: K8sActualSource
): WORKLOAD_PROGRAMMING_LANGUAGES => {
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
  const noneIgnoredLanguages = containers.filter(
    (container) => container.language !== 'ignored'
  );
  if (noneIgnoredLanguages.length === 0) {
    return WORKLOAD_PROGRAMMING_LANGUAGES.NO_CONTAINERS;
  }

  // find the first container with valid language
  const mainContainer = noneIgnoredLanguages.find(
    (container) => container.language !== 'unknown'
  );
  if (!mainContainer) {
    return WORKLOAD_PROGRAMMING_LANGUAGES.UNKNOWN; // no valid language found, return the first one
  }
  return mainContainer.language as WORKLOAD_PROGRAMMING_LANGUAGES;
};

export const getMainContainerLanguageLogo = (
  source: K8sActualSource
): string => {
  const language = getMainContainerLanguage(source);
  return LANGUAGES_LOGOS[language];
};

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
    [WORKLOAD_PROGRAMMING_LANGUAGES.NO_CONTAINERS]: '#000000',
  };

export const getMainContainerLanguage = (
  languages:
    | Array<{
        container_name: string;
        language: string;
      }>
    | undefined
): WORKLOAD_PROGRAMMING_LANGUAGES => {
  if (!languages) {
    return WORKLOAD_PROGRAMMING_LANGUAGES.PROCESSING;
  }

  // we will filter out the ignored languages as we don't want to account them in the main language
  const notIgnoredLanguages = languages?.filter(
    (container) => container.language !== 'ignored'
  );
  if (notIgnoredLanguages.length === 0) {
    return WORKLOAD_PROGRAMMING_LANGUAGES.NO_CONTAINERS;
  } else {
    // find the first container with valid language
    const mainContainer = languages?.find(
      (container) =>
        container.language !== 'default' && container.language !== 'unknown'
    );
    if (!mainContainer) {
      return languages[0].language as WORKLOAD_PROGRAMMING_LANGUAGES; // no valid language found, return the first one
    }
    return mainContainer.language as WORKLOAD_PROGRAMMING_LANGUAGES;
  }
};

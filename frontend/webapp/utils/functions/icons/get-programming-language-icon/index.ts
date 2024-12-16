import { WORKLOAD_PROGRAMMING_LANGUAGES } from '@/utils';
import { type SourceContainer } from '@/types';

export const getProgrammingLanguageIcon = (language: SourceContainer['language']) => {
  const BASE_URL = 'https://d1n7d4xz7fr8b4.cloudfront.net/';

  const LOGOS: Record<WORKLOAD_PROGRAMMING_LANGUAGES, string> = {
    [WORKLOAD_PROGRAMMING_LANGUAGES.JAVA]: `${BASE_URL}java.svg`,
    [WORKLOAD_PROGRAMMING_LANGUAGES.GO]: `${BASE_URL}go.svg`,
    [WORKLOAD_PROGRAMMING_LANGUAGES.JAVASCRIPT]: `${BASE_URL}nodejs.svg`,
    [WORKLOAD_PROGRAMMING_LANGUAGES.PYTHON]: `${BASE_URL}python.svg`,
    [WORKLOAD_PROGRAMMING_LANGUAGES.DOTNET]: `${BASE_URL}dotnet.svg`,
    [WORKLOAD_PROGRAMMING_LANGUAGES.MYSQL]: `${BASE_URL}mysql.svg`,
    [WORKLOAD_PROGRAMMING_LANGUAGES.NGINX]: `${BASE_URL}nginx.svg`,
    [WORKLOAD_PROGRAMMING_LANGUAGES.IGNORED]: '', // TODO: good icon
    [WORKLOAD_PROGRAMMING_LANGUAGES.UNKNOWN]: '', // TODO: good icon
    [WORKLOAD_PROGRAMMING_LANGUAGES.PROCESSING]: '', // TODO: good icon
    [WORKLOAD_PROGRAMMING_LANGUAGES.NO_CONTAINERS]: '', // TODO: good icon
    [WORKLOAD_PROGRAMMING_LANGUAGES.NO_RUNNING_PODS]: '', // TODO: good icon
  };

  return LOGOS[language];
};

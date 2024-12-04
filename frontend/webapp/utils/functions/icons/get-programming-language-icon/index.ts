import { WORKLOAD_PROGRAMMING_LANGUAGES } from '@/utils';
import { type SourceContainer } from '@/types';

const BRAND_ICON = '/brand/odigos-icon.svg';

export const getProgrammingLanguageIcon = (language?: SourceContainer['language']) => {
  if (!language) return BRAND_ICON;

  const BASE_URL = 'https://d1n7d4xz7fr8b4.cloudfront.net/';
  const LANGUAGES_LOGOS: Record<WORKLOAD_PROGRAMMING_LANGUAGES, string> = {
    [WORKLOAD_PROGRAMMING_LANGUAGES.JAVA]: `${BASE_URL}java.svg`,
    [WORKLOAD_PROGRAMMING_LANGUAGES.GO]: `${BASE_URL}go.svg`,
    [WORKLOAD_PROGRAMMING_LANGUAGES.JAVASCRIPT]: `${BASE_URL}nodejs.svg`,
    [WORKLOAD_PROGRAMMING_LANGUAGES.PYTHON]: `${BASE_URL}python.svg`,
    [WORKLOAD_PROGRAMMING_LANGUAGES.DOTNET]: `${BASE_URL}dotnet.svg`,
    [WORKLOAD_PROGRAMMING_LANGUAGES.MYSQL]: `${BASE_URL}mysql.svg`,
    [WORKLOAD_PROGRAMMING_LANGUAGES.NGINX]: `${BASE_URL}nginx.svg`,
    [WORKLOAD_PROGRAMMING_LANGUAGES.IGNORED]: BRAND_ICON, // TODO: good icon
    [WORKLOAD_PROGRAMMING_LANGUAGES.UNKNOWN]: BRAND_ICON, // TODO: good icon
    [WORKLOAD_PROGRAMMING_LANGUAGES.PROCESSING]: BRAND_ICON, // TODO: good icon
    [WORKLOAD_PROGRAMMING_LANGUAGES.NO_CONTAINERS]: BRAND_ICON, // TODO: good icon
    [WORKLOAD_PROGRAMMING_LANGUAGES.NO_RUNNING_PODS]: BRAND_ICON, // TODO: good icon
  };

  return LANGUAGES_LOGOS[language] || BRAND_ICON;
};

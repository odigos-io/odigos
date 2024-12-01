import { WORKLOAD_PROGRAMMING_LANGUAGES } from '../constants';
import { type ActionsType, type InstrumentationRuleType, type NotificationType, OVERVIEW_ENTITY_TYPES, SourceContainer } from '@/types';

const BRAND_ICON = '/brand/odigos-icon.svg';

export const getStatusIcon = (status?: NotificationType) => {
  if (!status) return BRAND_ICON;

  switch (status) {
    case 'success':
      return '/icons/notification/success-icon.svg';
    case 'error':
      return '/icons/notification/error-icon2.svg';
    case 'warning':
      return '/icons/notification/warning-icon2.svg';
    case 'info':
      return '/icons/common/info.svg';
    default:
      return BRAND_ICON;
  }
};

export const getEntityIcon = (type?: OVERVIEW_ENTITY_TYPES) => {
  if (!type) return BRAND_ICON;

  return `/icons/overview/${type}s.svg`;
};

export const getRuleIcon = (type?: InstrumentationRuleType) => {
  if (!type) return BRAND_ICON;

  const typeLowerCased = type.replaceAll('-', '').toLowerCase();

  return `/icons/rules/${typeLowerCased}.svg`;
};

export const getActionIcon = (type?: ActionsType | 'sampler' | 'attributes') => {
  if (!type) return BRAND_ICON;

  const typeLowerCased = type.toLowerCase();
  const isSampler = typeLowerCased.includes('sampler');
  const isAttributes = typeLowerCased === 'attributes';

  const iconName = isSampler ? 'sampler' : isAttributes ? 'piimasking' : typeLowerCased;

  return `/icons/actions/${iconName}.svg`;
};

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
    [WORKLOAD_PROGRAMMING_LANGUAGES.UNKNOWN]: `${BASE_URL}default.svg`, // TODO: good icon
    [WORKLOAD_PROGRAMMING_LANGUAGES.PROCESSING]: `${BASE_URL}default.svg`, // TODO: good icon
    [WORKLOAD_PROGRAMMING_LANGUAGES.NO_CONTAINERS]: `${BASE_URL}default.svg`, // TODO: good icon
    [WORKLOAD_PROGRAMMING_LANGUAGES.NO_RUNNING_PODS]: `${BASE_URL}default.svg`, // TODO: good icon
    [WORKLOAD_PROGRAMMING_LANGUAGES.NGINX]: `${BASE_URL}nginx.svg`,
  };

  return LANGUAGES_LOGOS[language] || BRAND_ICON;
};

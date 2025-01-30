import { K8sActualSource } from '@/types';
import { Types } from '@odigos/ui-components';

// while odigos lists language per container, we want to aggregate one single language for the workload.
// the process is mostly heuristic, we iterate over the containers and return the first valid language we find.
// there are additional cases for when the workload programming language is not available.

export const getMainContainerLanguage = (source: K8sActualSource): Types.PROGRAMMING_LANGUAGES => {
  const { numberOfInstances, containers } = source;

  if (!containers) {
    if (!!numberOfInstances && numberOfInstances > 0) {
      return Types.PROGRAMMING_LANGUAGES.PROCESSING;
    } else {
      return Types.PROGRAMMING_LANGUAGES.NO_RUNNING_PODS;
    }
  }

  // we will filter out the ignored languages as we don't want to account them in the main language
  const noneIgnoredLanguages = containers?.filter((container) => container.language !== Types.PROGRAMMING_LANGUAGES.IGNORED);
  if (!noneIgnoredLanguages.length) return Types.PROGRAMMING_LANGUAGES.NO_CONTAINERS;

  // find the first container with valid language
  const mainContainer = noneIgnoredLanguages.find((container) => container.language !== Types.PROGRAMMING_LANGUAGES.UNKNOWN);
  // no valid language found, return the first one
  if (!mainContainer) return Types.PROGRAMMING_LANGUAGES.UNKNOWN;

  return mainContainer.language;
};

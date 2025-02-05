import { ENTITY_TYPES } from '@odigos/ui-utils';
import { type Destination, type Action } from '@odigos/ui-containers';
import { type InstrumentationRuleSpec, type FetchedSource } from '@/types';

export type Category = 'all' | ENTITY_TYPES;

export const buildSearchResults = ({
  rules,
  sources,
  actions,
  destinations,
  searchText,
  selectedCategory,
}: {
  rules: InstrumentationRuleSpec[];
  sources: FetchedSource[];
  actions: Action[];
  destinations: Destination[];
  searchText: string;
  selectedCategory: Category;
}) => {
  const filteredRules = !searchText ? rules : rules.filter((rule) => rule.type?.toLowerCase().includes(searchText) || rule.ruleName?.toLowerCase().includes(searchText));
  const filteredSources = !searchText ? sources : sources.filter((source) => source.name?.toLowerCase().includes(searchText) || source.otelServiceName?.toLowerCase().includes(searchText));
  const filteredActions = !searchText ? actions : actions.filter((action) => action.type?.toLowerCase().includes(searchText) || action.spec.actionName?.toLowerCase().includes(searchText));
  const filteredDestinations = !searchText
    ? destinations
    : destinations.filter((destination) => destination.destinationType.displayName?.toLowerCase().includes(searchText) || destination.name?.toLowerCase().includes(searchText));

  const categories: {
    category: Category;
    label: string;
    count: number;
    entities: InstrumentationRuleSpec[] | FetchedSource[] | Action[] | Destination[];
  }[] = [
    {
      category: ENTITY_TYPES.SOURCE,
      label: 'Sources',
      count: filteredSources.length,
      entities: [],
    },
    {
      category: ENTITY_TYPES.ACTION,
      label: 'Actions',
      count: filteredActions.length,
      entities: [],
    },
    {
      category: ENTITY_TYPES.DESTINATION,
      label: 'Destinations',
      count: filteredDestinations.length,
      entities: [],
    },
    {
      category: ENTITY_TYPES.INSTRUMENTATION_RULE,
      label: 'Instrumentation Rules',
      count: filteredRules.length,
      entities: [],
    },
  ];

  categories.unshift({
    category: 'all',
    label: 'All',
    count: filteredRules.length + filteredSources.length + filteredActions.length + filteredDestinations.length,
    entities: [],
  });

  const searchResults = categories
    .filter(({ count, category }) => !!count && category !== 'all' && ['all', category].includes(selectedCategory))
    .map((item) => ({
      ...item,
      entities:
        item.category === ENTITY_TYPES.INSTRUMENTATION_RULE
          ? filteredRules
          : item.category === ENTITY_TYPES.SOURCE
          ? filteredSources
          : item.category === ENTITY_TYPES.ACTION
          ? filteredActions
          : item.category === ENTITY_TYPES.DESTINATION
          ? filteredDestinations
          : [],
    }));

  return { categories, searchResults };
};

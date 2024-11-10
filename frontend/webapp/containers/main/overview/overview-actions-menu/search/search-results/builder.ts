import { type ActionDataParsed, type ActualDestination, type InstrumentationRuleSpec, type K8sActualSource, OVERVIEW_ENTITY_TYPES } from '@/types';

export type Category = 'all' | OVERVIEW_ENTITY_TYPES;

export const buildFilteredLists = ({
  rules,
  sources,
  actions,
  destinations,
  searchText,
  selectedCategory,
}: {
  rules: InstrumentationRuleSpec[];
  sources: K8sActualSource[];
  actions: ActionDataParsed[];
  destinations: ActualDestination[];
  searchText: string;
  selectedCategory: Category;
}) => {
  const filteredRules = !searchText ? rules : rules.filter((rule) => rule.type?.toLowerCase().includes(searchText) || rule.ruleName.toLowerCase().includes(searchText));
  const filteredSources = !searchText ? sources : sources.filter((source) => source.name.toLowerCase().includes(searchText) || source.reportedName.toLowerCase().includes(searchText));
  const filteredActions = !searchText ? actions : actions.filter((action) => action.type.toLowerCase().includes(searchText) || action.spec.actionName.toLowerCase().includes(searchText));
  const filteredDestinations = !searchText
    ? destinations
    : destinations.filter((destination) => destination.destinationType.displayName.toLowerCase().includes(searchText) || destination.name.toLowerCase().includes(searchText));

  const categories: {
    category: Category;
    label: string;
    count: number;
    entities: InstrumentationRuleSpec[] | K8sActualSource[] | ActionDataParsed[] | ActualDestination[];
  }[] = [
    {
      category: OVERVIEW_ENTITY_TYPES.SOURCE,
      label: 'Sources',
      count: filteredSources.length,
      entities: [],
    },
    {
      category: OVERVIEW_ENTITY_TYPES.ACTION,
      label: 'Actions',
      count: filteredActions.length,
      entities: [],
    },
    {
      category: OVERVIEW_ENTITY_TYPES.DESTINATION,
      label: 'Destinations',
      count: filteredDestinations.length,
      entities: [],
    },
    {
      category: OVERVIEW_ENTITY_TYPES.RULE,
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
        item.category === OVERVIEW_ENTITY_TYPES.RULE
          ? filteredRules
          : item.category === OVERVIEW_ENTITY_TYPES.SOURCE
          ? filteredSources
          : item.category === OVERVIEW_ENTITY_TYPES.ACTION
          ? filteredActions
          : item.category === OVERVIEW_ENTITY_TYPES.DESTINATION
          ? filteredDestinations
          : [],
    }));

  return { categories, searchResults };
};

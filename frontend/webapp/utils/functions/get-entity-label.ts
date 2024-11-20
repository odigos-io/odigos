import { ActionData, ActionDataParsed, ActualDestination, InstrumentationRuleSpec, K8sActualSource, OVERVIEW_ENTITY_TYPES } from '@/types';

export const getEntityLabel = (
  entity: InstrumentationRuleSpec | K8sActualSource | ActionData | ActualDestination,
  entityType: OVERVIEW_ENTITY_TYPES,
  options?: { extended?: boolean; prioritizeDisplayName?: boolean },
): string => {
  const { extended, prioritizeDisplayName } = options || {};

  let type = '';
  let name = '';

  switch (entityType) {
    case OVERVIEW_ENTITY_TYPES.RULE:
      const rule = entity as InstrumentationRuleSpec;
      type = rule.type as string;
      name = rule.ruleName;
      break;

    case OVERVIEW_ENTITY_TYPES.SOURCE:
      const source = entity as K8sActualSource;
      type = source.name;
      name = source.reportedName;
      break;

    case OVERVIEW_ENTITY_TYPES.ACTION:
      const action = entity as ActionDataParsed;
      type = action.type;
      name = action.spec.actionName;
      break;

    case OVERVIEW_ENTITY_TYPES.DESTINATION:
      const destination = entity as ActualDestination;
      type = destination.destinationType.displayName;
      name = destination.name;
      break;

    default:
      break;
  }

  if (extended) return type + (name ? ` (${name})` : '');
  else if (prioritizeDisplayName) return name || type;
  else return type;
};

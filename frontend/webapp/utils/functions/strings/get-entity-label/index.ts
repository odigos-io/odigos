import { Types } from '@odigos/ui-components';
import { type ActionData, type ActionDataParsed, type ActualDestination, type InstrumentationRuleSpec, type K8sActualSource } from '@/types';

export const getEntityLabel = (
  entity: InstrumentationRuleSpec | K8sActualSource | ActionData | ActualDestination,
  entityType: Types.ENTITY_TYPES,
  options?: { extended?: boolean; prioritizeDisplayName?: boolean },
): string => {
  const { extended, prioritizeDisplayName } = options || {};

  let type = '';
  let name = '';

  switch (entityType) {
    case Types.ENTITY_TYPES.INSTRUMENTATION_RULE:
      const rule = entity as InstrumentationRuleSpec;
      type = rule.type as string;
      name = rule.ruleName;
      break;

    case Types.ENTITY_TYPES.SOURCE:
      const source = entity as K8sActualSource;
      type = source.name;
      name = source.otelServiceName;
      break;

    case Types.ENTITY_TYPES.ACTION:
      const action = entity as ActionDataParsed;
      type = action.type;
      name = action.spec.actionName;
      break;

    case Types.ENTITY_TYPES.DESTINATION:
      const destination = entity as ActualDestination;
      type = destination.destinationType.displayName;
      name = destination.name;
      break;

    default:
      break;
  }

  if (extended) return type + (name && name !== type ? ` (${name})` : '');
  else if (prioritizeDisplayName) return name || type;
  else return type;
};

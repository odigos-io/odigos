import { InstrumentationRuleType } from '@/types';
import { CodeAttributesIcon, OdigosLogo, PayloadCollectionIcon, Types } from '@odigos/ui-components';

export const getRuleIcon = (type: InstrumentationRuleType) => {
  const LOGOS: Record<InstrumentationRuleType, Types.SVG> = {
    [InstrumentationRuleType.PAYLOAD_COLLECTION]: PayloadCollectionIcon,
    [InstrumentationRuleType.CODE_ATTRIBUTES]: CodeAttributesIcon,
    [InstrumentationRuleType.UNKNOWN_TYPE]: OdigosLogo,
  };

  return LOGOS[type];
};

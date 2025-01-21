import { InstrumentationRuleType } from '@/types';
import { CodeAttributesIcon, OdigosLogo, PayloadCollectionIcon, SVG } from '@/assets';

export const getRuleIcon = (type: InstrumentationRuleType) => {
  const LOGOS: Record<InstrumentationRuleType, SVG> = {
    [InstrumentationRuleType.PAYLOAD_COLLECTION]: PayloadCollectionIcon,
    [InstrumentationRuleType.CODE_ATTRIBUTES]: CodeAttributesIcon,
    [InstrumentationRuleType.UNKNOWN_TYPE]: OdigosLogo,
  };

  return LOGOS[type];
};

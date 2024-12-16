import { PayloadCollectionIcon } from '@/assets';
import { InstrumentationRuleType } from '@/types';

export const getRuleIcon = (type?: InstrumentationRuleType) => {
  switch (type) {
    case InstrumentationRuleType.PAYLOAD_COLLECTION:
      return PayloadCollectionIcon;

    default:
      return undefined;
  }
};

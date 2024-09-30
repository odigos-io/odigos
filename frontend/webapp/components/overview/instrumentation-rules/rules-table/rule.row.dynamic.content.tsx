import { KeyvalText } from '@/design.system';
import theme from '@/styles/palette';
import { InstrumentationRuleSpec, RulesType } from '@/types';
import React from 'react';

interface RuleRowDynamicContentProps {
  item: InstrumentationRuleSpec;
}

export default function RuleRowDynamicContent({
  item,
}: RuleRowDynamicContentProps) {
  function renderContentByType() {
    switch ('payload-collection') {
      case RulesType.PAYLOAD_COLLECTION:
        return (
          <KeyvalText color={theme.text.grey} size={14} weight={400}>
            {' '}
          </KeyvalText>
        );

      default:
        return <div></div>;
    }
  }

  return <>{renderContentByType()}</>;
}

import React from 'react';
import { Funnel } from '@/assets';
import { OVERVIEW } from '@/utils';
import theme from '@/styles/palette';
import styled from 'styled-components';
import { KeyvalText } from '@/design.system';

const StyledThead = styled.div`
  background-color: ${theme.colors.light_dark};
  border-top-right-radius: 6px;
  border-top-left-radius: 6px;
`;

const StyledTh = styled.th`
  padding: 10px 20px;
  text-align: left;
  border-bottom: 1px solid ${theme.colors.blue_grey};
`;

const StyledMainTh = styled(StyledTh)`
  padding: 10px 20px;
  display: flex;
  align-items: center;
  gap: 8px;
`;

interface InstrumentationRulesTableHeaderProps {
  data: any[];
}

export function InstrumentationRulesTableHeader({
  data,
}: InstrumentationRulesTableHeaderProps) {
  return (
    <StyledThead>
      <StyledMainTh>
        <Funnel style={{ width: 18 }} />
        <KeyvalText size={14} weight={600} color={theme.text.white}>
          {`${data.length} ${OVERVIEW.MENU.INSTRUMENTATION_RULES}`}
        </KeyvalText>
      </StyledMainTh>
    </StyledThead>
  );
}

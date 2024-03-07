import React, { useEffect, useMemo, useState } from 'react';
import { OVERVIEW } from '@/utils';
import theme from '@/styles/palette';
import styled from 'styled-components';
import { UnFocusDestinations, UnFocusSources } from '@/assets/icons/side.menu';
import { ActionsGroup, KeyvalText } from '@/design.system';
import { Destination } from '@/types';

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

const ActionGroupContainer = styled.div`
  display: flex;
  justify-content: flex-end;
  padding-right: 20px;
  gap: 24px;
  flex: 1;
`;

interface DestinationsTableHeaderProps {
  data: Destination[];
}

export function DestinationsTableHeader({
  data,
}: DestinationsTableHeaderProps) {
  return (
    <StyledThead>
      <StyledMainTh>
        <UnFocusDestinations style={{ width: 18, height: 18 }} />
        <KeyvalText size={14} weight={600} color={theme.text.white}>
          {`${data.length} ${OVERVIEW.MENU.DESTINATIONS}`}
        </KeyvalText>
        <ActionGroupContainer>
          {/* <ActionsGroup actionGroups={sourcesGroups} /> */}
        </ActionGroupContainer>
      </StyledMainTh>
    </StyledThead>
  );
}

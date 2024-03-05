import React, { useMemo } from 'react';
import { ACTIONS } from '@/utils';
import theme from '@/styles/palette';
import { ActionData } from '@/types';
import { ACTION_ICONS } from '@/assets';
import styled, { css } from 'styled-components';
import { KeyvalCheckbox, KeyvalText } from '@/design.system';
import { TapList } from '@/components/setup/destination/tap.list/tap.list';
import { MONITORING_OPTIONS } from '@/components/setup/destination/utils';

const StyledTr = styled.tr`
  &:hover {
    background-color: ${theme.colors.light_dark};
  }
`;

const StyledTd = styled.td<{ isFirstRow?: boolean }>`
  padding: 10px 20px;
  border-top: 1px solid ${theme.colors.blue_grey};

  ${({ isFirstRow }) =>
    isFirstRow &&
    css`
      border-top: none;
    `}
`;

const StyledMainTd = styled(StyledTd)`
  cursor: pointer;
  padding: 10px 0px;
`;

const ActionIconContainer = styled.div`
  display: flex;
  gap: 8px;
`;

const ActionDetails = styled.div`
  display: flex;
  flex-direction: column;
  gap: 4px;
`;

const ClusterAttributesContainer = styled.div`
  display: flex;
  gap: 8px;
  align-items: center;
`;

const StatusIndicator = styled.div<{ disabled: boolean }>`
  width: 6px;
  height: 6px;
  border-radius: 4px;
  background-color: ${({ disabled }) =>
    disabled ? theme.colors.orange_brown : theme.colors.success};
`;

const TapListContainer = styled.div`
  padding: 0 20px;
  display: flex;
  align-items: center;
`;

const TAP_STYLE: React.CSSProperties = { padding: '4px 8px', gap: 4 };

const supported_signals = {
  traces: {
    supported: true,
  },
  metrics: {
    supported: true,
  },
  logs: {
    supported: true,
  },
};

export function ActionsTableRow({
  item,
  index,
  data,
  selectedCheckbox,
  onSelectedCheckboxChange,
  onRowClick,
}: {
  item: ActionData;
  index: number;
  data: ActionData[];
  selectedCheckbox: string[];
  onSelectedCheckboxChange: (id: string) => void;
  onRowClick: (id: string) => void;
}) {
  const ActionIcon = ACTION_ICONS[item.type];

  const monitors = useMemo(() => {
    return Object?.entries(supported_signals).reduce((acc, [key, _]) => {
      const monitor = MONITORING_OPTIONS.find(
        (option) => option.title.toLowerCase() === key
      );
      if (monitor && supported_signals[key].supported) {
        return [
          ...acc,
          {
            ...monitor,
            tapped: item.spec.signals.includes(key.toUpperCase()),
          },
        ];
      }

      return acc;
    }, []);
  }, [data]);

  return (
    <StyledTr key={item.id}>
      <StyledTd
        isFirstRow={index === 0}
        style={{
          display: 'flex',
          alignItems: 'flex-start',
          justifyContent: 'flex-start',
        }}
      >
        <KeyvalCheckbox
          value={selectedCheckbox.includes(item.id)}
          onChange={() => onSelectedCheckboxChange(item.id)}
        />
      </StyledTd>
      <StyledMainTd
        onClick={() => onRowClick(item.id)}
        isFirstRow={index === 0}
      >
        <ActionIconContainer>
          <div>
            <ActionIcon style={{ width: 16, height: 16 }} />
          </div>
          <ActionDetails>
            <KeyvalText color={theme.colors.light_grey} size={12}>
              {ACTIONS[item?.type || ''].TITLE}
            </KeyvalText>
            <ClusterAttributesContainer>
              <KeyvalText weight={600}>
                {`${item.spec.actionName || 'Action'} `}
              </KeyvalText>
              <StatusIndicator disabled={!!item.spec.disabled} />

              <KeyvalText color={theme.text.grey} size={14} weight={400}>
                {`${item?.spec.clusterAttributes.length} cluster attributes`}
              </KeyvalText>
            </ClusterAttributesContainer>
            <KeyvalText color={theme.text.light_grey} size={14}>
              {item.spec.notes}
            </KeyvalText>
          </ActionDetails>
          <TapListContainer>
            <TapList gap={4} list={monitors} tapStyle={TAP_STYLE} />
          </TapListContainer>
        </ActionIconContainer>
      </StyledMainTd>
    </StyledTr>
  );
}

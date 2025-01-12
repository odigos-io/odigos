import React, { useState } from 'react';
import { FlexRow } from '@/styles';
import styled from 'styled-components';
import { NOTIFICATION_TYPE } from '@/types';
import { DATA_CARDS, getStatusIcon, safeJsonStringify } from '@/utils';
import OverviewDrawer from '@/containers/main/overview/overview-drawer';
import { CodeBracketsIcon, CodeIcon, CopyIcon, KeyIcon, ListIcon } from '@/assets';
import { useComputePlatform, useCopy, useDescribeOdigos, useTimeAgo } from '@/hooks';
import { DataCard, DataCardFieldTypes, IconButton, Segment } from '@/reuseable-components';

interface Props {}

const DataContainer = styled.div`
  display: flex;
  flex-direction: column;
  gap: 12px;
`;

export const CliDrawer: React.FC<Props> = () => {
  const timeAgo = useTimeAgo();
  const { data: cp } = useComputePlatform();
  const { isCopied, copiedIndex, clickCopy } = useCopy();
  const { data: describe, restructureForPrettyMode } = useDescribeOdigos();

  const [isPrettyMode, setIsPrettyMode] = useState(true);

  const tokens = cp?.computePlatform.apiTokens || [];

  return (
    <OverviewDrawer title='Odigos CLI' icon={CodeBracketsIcon}>
      <DataContainer>
        {!!tokens?.length && (
          <DataCard
            title={DATA_CARDS.API_TOKENS}
            titleBadge={tokens.length}
            data={[
              {
                type: DataCardFieldTypes.TABLE,
                value: {
                  columns: [
                    { key: 'icon', title: '' },
                    { key: 'name', title: 'Name' },
                    { key: 'expires_at', title: 'Expires' },
                    { key: 'token', title: 'Token' },
                    { key: 'actions', title: '' },
                  ],
                  rows: tokens.map(({ token, name, expiresAt }, idx) => [
                    { columnKey: 'icon', icon: KeyIcon },
                    { columnKey: 'name', value: name },
                    { columnKey: 'expires_at', value: `${timeAgo.format(expiresAt)} (${new Date(expiresAt).toDateString().split(' ').slice(1).join(' ')})` },
                    { columnKey: 'token', value: `${new Array(15).fill('•').join('')}` },
                    {
                      columnKey: 'actions',
                      component: () => {
                        const SuccessIcon = getStatusIcon(NOTIFICATION_TYPE.SUCCESS);

                        return (
                          <FlexRow $gap={0}>
                            <IconButton size={32} onClick={() => clickCopy(token, idx)}>
                              {isCopied && copiedIndex === idx ? <SuccessIcon /> : <CopyIcon />}
                            </IconButton>

                            {/* <Divider orientation='vertical' length='12px' />

                            <IconButton size={32} onClick={() => {}}>
                              <EditIcon />
                            </IconButton> */}
                          </FlexRow>
                        );
                      },
                    },
                  ]),
                },
                width: 'inherit',
              },
            ]}
          />
        )}

        <DataCard
          title={DATA_CARDS.DESCRIBE_ODIGOS}
          action={
            <Segment
              options={[
                { icon: ListIcon, value: true },
                { icon: CodeIcon, value: false },
              ]}
              selected={isPrettyMode}
              setSelected={setIsPrettyMode}
            />
          }
          data={[
            {
              type: DataCardFieldTypes.CODE,
              value: JSON.stringify({
                language: 'json',
                code: safeJsonStringify(isPrettyMode ? restructureForPrettyMode(describe) : describe),
                pretty: isPrettyMode,
              }),
              width: 'inherit',
            },
          ]}
        />
      </DataContainer>
    </OverviewDrawer>
  );
};

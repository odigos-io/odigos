import React, { useState } from 'react';
import { FlexRow } from '@/styles';
import styled from 'styled-components';
import { NOTIFICATION_TYPE } from '@/types';
import { useNotificationStore } from '@/store';
import { useApiTokens, useCopy, useDescribeOdigos } from '@/hooks';
import { DATA_CARDS, getStatusIcon, safeJsonStringify } from '@/utils';
import OverviewDrawer from '@/containers/main/overview/overview-drawer';
import { DataCard, DataCardFieldTypes, Divider, IconButton, Segment } from '@/reuseable-components';
import { CodeBracketsIcon, CodeIcon, CopyIcon, EditIcon, KeyIcon, ListIcon, TrashIcon } from '@/assets';

interface Props {}

const DataContainer = styled.div`
  display: flex;
  flex-direction: column;
  gap: 12px;
`;

export const CliDrawer: React.FC<Props> = () => {
  const { data: tokens } = useApiTokens();
  const { isCopied, copiedIndex, clickCopy } = useCopy();
  const { data: describe, restructureForPrettyMode } = useDescribeOdigos();

  const [isPrettyMode, setIsPrettyMode] = useState(true);

  // temporary until we have edit & delete for tokens
  const { addNotification } = useNotificationStore();

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
                    { key: 'created_at', title: 'Created' },
                    { key: 'expires_at', title: 'Expires' },
                    { key: 'token', title: 'Token' },
                    { key: 'actions', title: '' },
                  ],
                  rows: tokens.map(({ token, aud, iat, exp }, idx) => [
                    { columnKey: 'icon', icon: KeyIcon },
                    { columnKey: 'name', value: aud },
                    { columnKey: 'created_at', value: new Date(iat).toLocaleDateString('en-US') },
                    { columnKey: 'expires_at', value: new Date(exp).toLocaleDateString('en-US') },
                    { columnKey: 'token', value: `${new Array(15).fill('•').join('')}` },
                    {
                      columnKey: 'actions',
                      component: () => (
                        <FlexRow $gap={0}>
                          <IconButton size={32} onClick={() => clickCopy(token, idx)}>
                            {isCopied && copiedIndex === idx ? getStatusIcon(NOTIFICATION_TYPE.SUCCESS)({}) : <CopyIcon />}
                          </IconButton>

                          <Divider orientation='vertical' length='12px' />

                          <IconButton size={32} onClick={() => addNotification({ type: NOTIFICATION_TYPE.WARNING, message: 'Coming soon!' })}>
                            <EditIcon />
                          </IconButton>

                          <Divider orientation='vertical' length='12px' />

                          <IconButton size={32} onClick={() => addNotification({ type: NOTIFICATION_TYPE.WARNING, message: 'Coming soon!' })}>
                            <TrashIcon />
                          </IconButton>
                        </FlexRow>
                      ),
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

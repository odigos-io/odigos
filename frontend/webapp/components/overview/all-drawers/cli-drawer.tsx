import React, { useRef, useState } from 'react';
import Theme from '@odigos/ui-theme';
import styled from 'styled-components';
import { useDrawerStore } from '@odigos/ui-containers';
import { DATA_CARDS, SEVEN_DAYS_IN_MS } from '@/utils';
import { useDescribeOdigos, useTokenCRUD } from '@/hooks';
import { CheckIcon, CodeBracketsIcon, CodeIcon, CopyIcon, CrossIcon, EditIcon, KeyIcon, ListIcon } from '@odigos/ui-icons';
import { getStatusIcon, isOverTime, NOTIFICATION_TYPE, safeJsonStringify, useCopy, useKeyDown, useOnClickOutside, useTimeAgo } from '@odigos/ui-utils';
import { Button, DATA_CARD_FIELD_TYPES, DataCard, Divider, Drawer, FlexColumn, FlexRow, IconButton, Input, Segment, Text, Tooltip } from '@odigos/ui-components';

interface Props {}

const DataContainer = styled.div`
  display: flex;
  flex-direction: column;
  gap: 12px;
`;

const Relative = styled.div`
  position: relative;
`;

const TokenPopover = styled(FlexColumn)`
  position: absolute;
  top: 32px;
  right: 0;
  z-index: 1;
  gap: 8px;
  padding: 24px;
  background-color: ${({ theme }) => theme.colors.info};
  border: 1px solid ${({ theme }) => theme.colors.border};
  border-radius: 24px;
`;

const PopoverFormWrapper = styled(FlexRow)`
  width: 100%;
`;

const PopoverFormButton = styled(Button)`
  width: 36px;
  padding-left: 0;
  padding-right: 0;
`;

const DRAWER_WIDTH = '750px';

export const CliDrawer: React.FC<Props> = () => {
  const theme = Theme.useTheme();
  const timeAgo = useTimeAgo();
  const { setDrawerType } = useDrawerStore();
  const { isCopied, copiedIndex, clickCopy } = useCopy();
  const { tokens, loading, updateToken } = useTokenCRUD();
  const { data: describe, restructureForPrettyMode } = useDescribeOdigos();

  const [isPrettyMode, setIsPrettyMode] = useState(true);
  const [editTokenIndex, setEditTokenIndex] = useState(-1);

  const tokenPopoverRef = useRef<HTMLDivElement>(null);
  const tokenInputRef = useRef<HTMLInputElement>(null);
  useOnClickOutside(tokenPopoverRef, () => setEditTokenIndex(-1));
  useKeyDown({ key: 'Enter', active: editTokenIndex !== -1 }, saveToken);

  function saveToken() {
    const token = tokenInputRef.current?.value;
    if (token) updateToken(token).then(() => setEditTokenIndex(-1));
  }

  const closeDrawer = () => {
    setDrawerType(null);
  };

  return (
    <Drawer
      isOpen={true}
      onClose={closeDrawer}
      width={DRAWER_WIDTH}
      header={{
        icon: CodeBracketsIcon,
        title: 'Odigos CLI',
      }}
      footer={{
        isOpen: false,
      }}
    >
      <DataContainer>
        {!!tokens?.length && (
          <DataCard
            title={DATA_CARDS.API_TOKENS}
            titleBadge={tokens.length}
            data={[
              {
                type: DATA_CARD_FIELD_TYPES.TABLE,
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
                    { columnKey: 'token', value: `${new Array(15).fill('•').join('')}` },
                    {
                      columnKey: 'expires_at',
                      component: () => {
                        return (
                          <Text size={14} color={isOverTime(expiresAt, 0) ? theme.text.error : isOverTime(expiresAt, SEVEN_DAYS_IN_MS) ? theme.text.warning : theme.text.success}>
                            {timeAgo.format(expiresAt)} ({new Date(expiresAt).toDateString().split(' ').slice(1).join(' ')})
                          </Text>
                        );
                      },
                    },
                    {
                      columnKey: 'actions',
                      component: () => {
                        const SuccessIcon = getStatusIcon(NOTIFICATION_TYPE.SUCCESS);

                        return (
                          <FlexRow $gap={0}>
                            <IconButton size={32} onClick={() => clickCopy(token, idx)}>
                              {isCopied && copiedIndex === idx ? <SuccessIcon /> : <CopyIcon />}
                            </IconButton>
                            <Divider orientation='vertical' length='12px' />

                            <Relative>
                              <IconButton size={32} onClick={() => setEditTokenIndex(idx)}>
                                <EditIcon />
                              </IconButton>

                              {editTokenIndex === idx && (
                                <TokenPopover ref={tokenPopoverRef}>
                                  <Tooltip text='Contact us to generate a new one' withIcon>
                                    <Text size={14} style={{ lineHeight: '20px', display: 'flex' }}>
                                      Enter a new API Token:
                                    </Text>
                                  </Tooltip>
                                  <PopoverFormWrapper>
                                    <Input ref={tokenInputRef} placeholder='API Token' autoFocus />
                                    <PopoverFormButton variant='primary' disabled={loading} onClick={saveToken}>
                                      <CheckIcon fill={theme.text.primary} />
                                    </PopoverFormButton>
                                    <PopoverFormButton variant='secondary' disabled={loading} onClick={() => setEditTokenIndex(-1)}>
                                      <CrossIcon />
                                    </PopoverFormButton>
                                  </PopoverFormWrapper>
                                </TokenPopover>
                              )}
                            </Relative>
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
              type: DATA_CARD_FIELD_TYPES.CODE,
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
    </Drawer>
  );
};

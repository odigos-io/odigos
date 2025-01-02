import React, { useMemo, useState } from 'react';
import { slide } from '@/styles';
import theme from '@/styles/theme';
import { TrashIcon } from '@/assets';
import { useAppStore } from '@/store';
import styled from 'styled-components';
import { DeleteWarning } from '@/components';
import { OVERVIEW_ENTITY_TYPES } from '@/types';
import { useSourceCRUD, useTransition } from '@/hooks';
import { Badge, Button, Divider, Text } from '@/reuseable-components';

const Container = styled.div`
  position: fixed;
  bottom: 0;
  left: 50%;
  transform: translate(-50%, 100%);
  z-index: 1000;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 24px;
  border-radius: 32px;
  border: 1px solid ${({ theme }) => theme.colors.border};
  background-color: ${({ theme }) => theme.colors.dropdown_bg};
`;

export const MultiSourceControl = () => {
  const Transition = useTransition({
    container: Container,
    animateIn: slide.in['center'],
    animateOut: slide.out['center'],
  });

  const { sources, persistSources } = useSourceCRUD();
  const { configuredSources, setConfiguredSources } = useAppStore();
  const [isWarnModalOpen, setIsWarnModalOpen] = useState(false);

  const totalSelected = useMemo(() => {
    let num = 0;

    Object.values(configuredSources).forEach((selectedSources) => {
      num += selectedSources.length;
    });

    return num;
  }, [configuredSources]);

  const onDeselect = () => {
    setConfiguredSources({});
  };

  const onDelete = () => {
    const payload = {};

    Object.entries(configuredSources).forEach(([namespace, sources]) => {
      payload[namespace] = sources.map(({ name, kind }) => ({ name, kind, selected: false }));
    });

    persistSources(payload, {});
    setIsWarnModalOpen(false);
    onDeselect();
  };

  return (
    <>
      <Transition data-id='multi-source-control' enter={!!totalSelected}>
        <Text>Selected sources</Text>
        <Badge label={totalSelected} filled />

        <Divider orientation='vertical' length='16px' />

        <Button variant='tertiary' onClick={onDeselect}>
          <Text family='secondary' decoration='underline'>
            Deselect
          </Text>
        </Button>

        <Button variant='tertiary' onClick={() => setIsWarnModalOpen(true)}>
          <TrashIcon />
          <Text family='secondary' decoration='underline' color={theme.text.error}>
            Uninstrument
          </Text>
        </Button>
      </Transition>

      <DeleteWarning
        isOpen={isWarnModalOpen}
        name={`${totalSelected} sources`}
        type={OVERVIEW_ENTITY_TYPES.SOURCE}
        isLastItem={totalSelected === sources.length}
        onApprove={onDelete}
        onDeny={() => setIsWarnModalOpen(false)}
      />
    </>
  );
};

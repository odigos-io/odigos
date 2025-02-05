import React, { useMemo, useState } from 'react';
import { useSourceCRUD } from '@/hooks';
import { Theme } from '@odigos/ui-theme';
import { TrashIcon } from '@odigos/ui-icons';
import { type FetchedSource } from '@/types';
import styled, { useTheme } from 'styled-components';
import { useSelectedStore } from '@odigos/ui-containers';
import { ENTITY_TYPES, useTransition } from '@odigos/ui-utils';
import { Badge, Button, DeleteWarning, Divider, Text } from '@odigos/ui-components';

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

const MultiSourceControl = () => {
  const Transition = useTransition({
    container: Container,
    animateIn: Theme.slide.in['center'],
    animateOut: Theme.slide.out['center'],
  });

  const theme = useTheme();
  const { sources, persistSources } = useSourceCRUD();
  const { selectedSources, setSelectedSources } = useSelectedStore();
  const [isWarnModalOpen, setIsWarnModalOpen] = useState(false);

  const totalSelected = useMemo(() => {
    let num = 0;

    Object.values(selectedSources).forEach((sources) => {
      num += sources.length;
    });

    return num;
  }, [selectedSources]);

  const onDeselect = () => {
    setSelectedSources({});
  };

  const onDelete = () => {
    const payload: Record<string, FetchedSource[]> = {};

    Object.entries(selectedSources).forEach(([namespace, sources]: [string, FetchedSource[]]) => {
      payload[namespace] = sources.map((source) => ({ ...source, selected: false }));
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
        type={ENTITY_TYPES.SOURCE}
        isLastItem={totalSelected === sources.length}
        onApprove={onDelete}
        onDeny={() => setIsWarnModalOpen(false)}
      />
    </>
  );
};

export default MultiSourceControl;

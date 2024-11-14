import React, { useState } from 'react';
import { useAppStore } from '@/store';
import styled from 'styled-components';
import { Badge, Button, Divider, Text } from '@/reuseable-components';
import { useSourceCRUD } from '@/hooks';
import theme from '@/styles/theme';
import Image from 'next/image';
import { slide } from '@/styles';
import { DeleteWarning } from '@/components';

const Container = styled.div<{ isEntering: boolean; isLeaving: boolean }>`
  position: fixed;
  bottom: 0;
  left: 50%;
  transform: translateX(-50%);
  z-index: 1000;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 24px;
  border-radius: 32px;
  border: 1px solid ${({ theme }) => theme.colors.border};
  background-color: ${({ theme }) => theme.colors.dropdown_bg};
  animation: ${({ isEntering, isLeaving }) => (isEntering ? slide.in['center'] : isLeaving ? slide.out['center'] : 'none')} 0.3s forwards;
`;

const MultiSourceControl = () => {
  const { sources, deleteSources } = useSourceCRUD();
  const { configuredSources, setConfiguredSources } = useAppStore((state) => state);
  const [isWarnModalOpen, setIsWarnModalOpen] = useState(false);

  const { namespace } = sources[0] || {};
  const count = !!configuredSources[namespace] ? configuredSources[namespace].length : 0;

  const onDeselect = () => {
    setConfiguredSources({});
  };

  const onDelete = () => {
    deleteSources(configuredSources);
    onDeselect();
    setIsWarnModalOpen(false);
  };

  return (
    <>
      <Container isEntering={!!count} isLeaving={!count}>
        <Text>Selected sources</Text>
        <Badge label={count} filled />

        <Divider orientation='vertical' length='16px' />

        <Button variant='tertiary' onClick={onDeselect}>
          <Text family='secondary' decoration='underline'>
            Deselect
          </Text>
        </Button>

        <Button variant='tertiary' onClick={() => setIsWarnModalOpen(true)}>
          <Image src='/icons/common/trash.svg' alt='' width={16} height={16} />
          <Text family='secondary' decoration='underline' color={theme.text.error}>
            Delete
          </Text>
        </Button>
      </Container>

      <DeleteWarning isOpen={isWarnModalOpen} name={`${count} sources`} onApprove={onDelete} onDeny={() => setIsWarnModalOpen(false)} />
    </>
  );
};

export default MultiSourceControl;

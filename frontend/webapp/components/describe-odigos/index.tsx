import React from 'react';
import Image from 'next/image';
import theme from '@/styles/theme';
import { IconButton } from '@/reuseable-components';
import { DRAWER_OTHER_TYPES, useDrawerStore } from '@/store';

export const DescribeOdigos = () => {
  const { setSelectedItem } = useDrawerStore();
  const handleClick = () => setSelectedItem({ type: DRAWER_OTHER_TYPES.DESCRIBE_ODIGOS, id: DRAWER_OTHER_TYPES.DESCRIBE_ODIGOS });

  return (
    <IconButton onClick={handleClick} tooltip='Describe Odigos' withPing pingColor={theme.colors.majestic_blue}>
      <Image src='/brand/odigos-icon.svg' alt='logo' width={16} height={16} />
    </IconButton>
  );
};

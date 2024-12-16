import React from 'react';
import theme from '@/styles/theme';
import { OdigosLogo } from '@/assets';
import { IconButton } from '@/reuseable-components';
import { DRAWER_OTHER_TYPES, useDrawerStore } from '@/store';

export const DescribeOdigos = () => {
  const { setSelectedItem } = useDrawerStore();
  const handleClick = () => setSelectedItem({ type: DRAWER_OTHER_TYPES.DESCRIBE_ODIGOS, id: DRAWER_OTHER_TYPES.DESCRIBE_ODIGOS });

  return (
    <IconButton onClick={handleClick} tooltip='Describe Odigos' withPing pingColor={theme.colors.majestic_blue}>
      <OdigosLogo size={12} />
    </IconButton>
  );
};

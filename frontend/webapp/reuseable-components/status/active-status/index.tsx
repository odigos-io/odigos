import React from 'react';
import { Status, type StatusProps } from '@/reuseable-components';

interface Props extends StatusProps {}

export const ActiveStatus: React.FC<Props> = ({ isActive, ...props }) => {
  return <Status title={isActive ? 'Active' : 'Inactive'} isActive={isActive} {...props} />;
};

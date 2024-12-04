import React from 'react';
import { Status, type StatusProps } from '@/reuseable-components';

interface Props extends StatusProps {}

export const ConnectionStatus: React.FC<Props> = ({ ...props }) => {
  return <Status size={14} family='primary' withIcon withBackground {...props} />;
};

import React from 'react';
import Image from 'next/image';
import { capitalizeFirstLetter } from '@/utils';
import { Gap, Tooltip } from '@/reuseable-components';

interface Props {
  monitors: string[];
  withTooltips?: boolean;
  size?: number;
}

export const MonitorsIcons: React.FC<Props> = ({ monitors, withTooltips, size = 12 }) => {
  return (
    <Gap size={size / 3}>
      {monitors.map((str) => {
        const signal = str.toLocaleLowerCase();

        return (
          <Tooltip key={signal} text={withTooltips ? capitalizeFirstLetter(signal) : ''}>
            <Image src={`/icons/monitors/${signal}.svg`} alt={signal} width={size} height={size} />
          </Tooltip>
        );
      })}
    </Gap>
  );
};

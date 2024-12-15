import React from 'react';
import Image from 'next/image';
import theme from '@/styles/theme';
import { FlexRow } from '@/styles';
import { capitalizeFirstLetter } from '@/utils';
import { Text, Tooltip } from '@/reuseable-components';

interface Props {
  monitors: string[];
  withTooltips?: boolean;
  withLabels?: boolean;
  size?: number;
}

export const MonitorsIcons: React.FC<Props> = ({ monitors, withTooltips, withLabels, size = 12 }) => {
  return (
    <FlexRow $gap={size / 2}>
      {monitors.map((str) => {
        const signal = str.toLowerCase();
        const signalDisplayName = capitalizeFirstLetter(signal);

        return (
          <Tooltip key={signal} text={withTooltips ? signalDisplayName : ''}>
            <FlexRow>
              <Image src={`/icons/monitors/${signal}.svg`} alt={signal} width={size} height={size} />
              {withLabels && (
                <Text size={size} color={theme.text.grey}>
                  {signalDisplayName}
                </Text>
              )}
            </FlexRow>
          </Tooltip>
        );
      })}
    </FlexRow>
  );
};

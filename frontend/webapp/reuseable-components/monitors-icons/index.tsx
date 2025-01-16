import React from 'react';
import theme from '@/styles/theme';
import { FlexRow } from '@/styles';
import { Text, Tooltip } from '@/reuseable-components';
import { capitalizeFirstLetter, getMonitorIcon, MONITORS_OPTIONS } from '@/utils';

interface Props {
  monitors?: string[];
  withTooltips?: boolean;
  withLabels?: boolean;
  size?: number;
  color?: string;
}

const defaultMonitors = MONITORS_OPTIONS.map(({ id }) => id);

export const MonitorsIcons: React.FC<Props> = ({ monitors = defaultMonitors, withTooltips, withLabels, size = 12, color = theme.text.grey }) => {
  return (
    <FlexRow $gap={withLabels ? size : size / 2}>
      {monitors
        .filter((str) => !!str)
        .map((str) => {
          const signal = str.toLowerCase();
          const signalDisplayName = capitalizeFirstLetter(signal);
          const Icon = getMonitorIcon(signal);

          return (
            <Tooltip key={signal} text={withTooltips ? signalDisplayName : ''}>
              <FlexRow $gap={size / 3}>
                <Icon size={withLabels ? size + 2 : size} fill={color} />

                {withLabels && (
                  <Text size={size} color={color}>
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

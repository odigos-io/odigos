import { CheckCircledIcon, ErrorTriangleIcon, InfoIcon, OdigosLogo, SVG, WarningTriangleIcon } from '@/assets';
import { NOTIFICATION_TYPE } from '@/types';
import { useTheme } from 'styled-components';

export const getStatusIcon = (type: NOTIFICATION_TYPE) => {
  const theme = useTheme();

  const LOGOS: Record<NOTIFICATION_TYPE, SVG> = {
    [NOTIFICATION_TYPE.SUCCESS]: () => CheckCircledIcon({ fill: theme.text.success }),
    [NOTIFICATION_TYPE.ERROR]: ErrorTriangleIcon,
    [NOTIFICATION_TYPE.WARNING]: WarningTriangleIcon,
    [NOTIFICATION_TYPE.INFO]: InfoIcon,
    [NOTIFICATION_TYPE.DEFAULT]: OdigosLogo,
  };

  return LOGOS[type];
};

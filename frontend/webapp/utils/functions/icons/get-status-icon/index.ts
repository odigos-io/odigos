import { ErrorTriangleIcon, SuccessRoundIcon, SVG, WarningTriangleIcon } from '@/assets';
import { NOTIFICATION_TYPE } from '@/types';

export const getStatusIcon = (type: NOTIFICATION_TYPE) => {
  const LOGOS: Record<NOTIFICATION_TYPE, SVG> = {
    [NOTIFICATION_TYPE.SUCCESS]: SuccessRoundIcon,
    [NOTIFICATION_TYPE.ERROR]: ErrorTriangleIcon,
    [NOTIFICATION_TYPE.WARNING]: WarningTriangleIcon,
    [NOTIFICATION_TYPE.INFO]: WarningTriangleIcon,
    [NOTIFICATION_TYPE.DEFAULT]: WarningTriangleIcon,
  };

  return LOGOS[type];
};

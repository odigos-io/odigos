const ICON_PATH = '/icons/notification/';

export const getStatusIcon = (active?: boolean) => {
  return `${ICON_PATH}${active ? 'success-icon' : 'error-icon2'}.svg`;
};

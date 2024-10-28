const ACTION_ICON_PATH = '/icons/actions/';

const getActionIcon = (actionType?: string) => {
  if (!actionType) {
    return `${ACTION_ICON_PATH}add-action.svg`;
  }

  const typeLowerCased = actionType.toLowerCase();
  const isSampler = typeLowerCased.includes('sampler');

  return `${ACTION_ICON_PATH}${isSampler ? 'sampler' : typeLowerCased}.svg`;
};

export default getActionIcon;

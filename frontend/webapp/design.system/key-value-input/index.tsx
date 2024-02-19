import React from 'react';

import { KeyValueTable, KeyValueTableProps } from '@keyval-dev/design-system';

export const KeyValuePair: React.FC<KeyValueTableProps> = (
  props: KeyValueTableProps
) => {
  return <KeyValueTable {...props} />;
};

import React from 'react';

import { KeyValueTable, KeyValueTableProps } from '@odigos-io/design-system';

export const KeyValuePair: React.FC<KeyValueTableProps> = (
  props: KeyValueTableProps
) => {
  return <KeyValueTable {...props} />;
};

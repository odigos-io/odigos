import React from 'react';
import { MultiInput } from '@keyval-dev/design-system';

interface MultiInputProps {
  initialList?: string[];
  onListChange?: (list: string[]) => void;
  placeholder?: string;
  limit?: number;
  tooltip?: string;
  title?: string;
}

export function KeyvalMultiInput(props: MultiInputProps): JSX.Element {
  return <MultiInput {...props} />;
}

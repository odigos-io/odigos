import React from 'react';
import { DropDown } from '@odigos-io/design-system';

interface DropDownItem {
  id: number;
  label: string;
}
interface KeyvalDropDownProps {
  data: DropDownItem[];
  onChange: (item: DropDownItem) => void;
  width?: number;
  value?: DropDownItem | null;
  label?: string;
  tooltip?: string;
  required?: boolean;
}

export function KeyvalDropDown(props: KeyvalDropDownProps) {
  return <DropDown {...props} />;
}

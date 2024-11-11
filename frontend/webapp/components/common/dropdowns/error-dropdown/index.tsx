import React, { useMemo } from 'react';
// import { useSourceCRUD } from '@/hooks';
import { DropdownOption } from '@/types';
import { Dropdown } from '@/reuseable-components';

interface Props {
  value?: DropdownOption[];
  onSelect: (val: DropdownOption) => void;
  onDeselect: (val: DropdownOption) => void;
  isMulti?: boolean;
  required?: boolean;
}

export const ErrorDropdown: React.FC<Props> = ({ value, onSelect, onDeselect, ...props }) => {
  // const { sources } = useSourceCRUD();

  const options = useMemo(() => {
    const payload: DropdownOption[] = [];

    // TODO: pull errors from sources
    // sources.forEach(({ ... }) => {
    //   if (!payload.find((opt) => opt.id === ...)) payload.push({ id: ..., value: ..., });
    // });

    return payload;
  }, []);

  return <Dropdown title='Error Message' placeholder='All' options={options} value={value} onSelect={onSelect} onDeselect={onDeselect} showSearch={false} {...props} />;
};

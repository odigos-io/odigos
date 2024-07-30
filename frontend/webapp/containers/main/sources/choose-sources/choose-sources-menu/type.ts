import { DropdownOption } from '@/types';

export type ToggleCheckboxState = {
  selectedAppsCount: number;
  selectAllCheckbox: boolean;
  showSelectedOnly: boolean;
  futureAppsCheckbox: boolean;
};

export type ToggleCheckboxHandlers = {
  setSelectAllCheckbox: (value: boolean) => void;
  setShowSelectedOnly: (value: boolean) => void;
  setFutureAppsCheckbox: (value: boolean) => void;
};

export type SearchDropdownState = {
  selectedOption: DropdownOption | undefined;
  searchFilter: string;
};

export type SearchDropdownHandlers = {
  setSelectedOption: (option: DropdownOption) => void;
  setSearchFilter: (search: string) => void;
};

export type SearchDropdownProps = {
  state: SearchDropdownState;
  handlers: SearchDropdownHandlers;
  dropdownOptions: DropdownOption[];
};

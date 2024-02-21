import React, { useEffect, useState } from 'react';
import { CheckboxWrapper, SwitcherWrapper } from './sources.option.menu.styled';
import { KeyvalCheckbox, KeyvalSwitch, KeyvalTooltip } from '@/design.system';
import { SETUP } from '@/utils/constants';

export function ActionSourcesOptions({
  onSelectAllChange,
  selectedApplications,
  currentNamespace,
  onFutureApplyChange,
}: any) {
  const [checked, setChecked] = useState(false);
  const [toggle, setToggle] = useState(false);

  useEffect(() => {
    setToggle(selectedApplications[currentNamespace?.name]?.selected_all);
    setChecked(selectedApplications[currentNamespace?.name]?.future_selected);
  }, [currentNamespace, selectedApplications]);

  const handleToggleChange = () => {
    onSelectAllChange(!toggle);
    setToggle(!toggle);
  };

  return (
    <>
      <SwitcherWrapper>
        <KeyvalSwitch
          label={SETUP.MENU.SELECT_ALL}
          toggle={toggle}
          handleToggleChange={handleToggleChange}
        />
      </SwitcherWrapper>
      <CheckboxWrapper>
        <KeyvalTooltip text={SETUP.MENU.TOOLTIP}>
          <KeyvalCheckbox
            label={SETUP.MENU.FUTURE_APPLY}
            value={checked}
            onChange={() => onFutureApplyChange(!checked)}
            disabled={!toggle}
          />
        </KeyvalTooltip>
      </CheckboxWrapper>
    </>
  );
}

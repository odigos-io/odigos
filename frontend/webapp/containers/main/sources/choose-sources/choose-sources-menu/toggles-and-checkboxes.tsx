import React from 'react';
import styled from 'styled-components';
import { Counter, Toggle, Checkbox } from '@/reuseable-components';
import { ToggleCheckboxHandlers, ToggleCheckboxState } from './type';

const Container = styled.div`
  display: flex;
  justify-content: space-between;
  align-items: center;
`;

const ToggleWrapper = styled.div`
  display: flex;
  align-items: center;
  gap: 32px;
`;

type ToggleCheckboxProps = {
  state: ToggleCheckboxState;
  handlers: ToggleCheckboxHandlers;
};

const TogglesAndCheckboxes: React.FC<ToggleCheckboxProps> = ({
  state,
  handlers,
}) => {
  const {
    selectedAppsCount,
    selectAllCheckbox,
    showSelectedOnly,
    futureAppsCheckbox,
  } = state;

  const { setSelectAllCheckbox, setShowSelectedOnly, setFutureAppsCheckbox } =
    handlers;
  return (
    <Container>
      <Counter value={selectedAppsCount} title="Selected apps" />
      <ToggleWrapper>
        <Toggle
          title="Select all"
          initialValue={selectAllCheckbox}
          onChange={setSelectAllCheckbox}
        />
        <Toggle
          title="Show selected only"
          initialValue={showSelectedOnly}
          onChange={setShowSelectedOnly}
        />
        <Checkbox
          title="Future apps"
          tooltip="Automatically instrument all future apps"
          initialValue={futureAppsCheckbox}
          onChange={setFutureAppsCheckbox}
        />
      </ToggleWrapper>
    </Container>
  );
};

export { TogglesAndCheckboxes };

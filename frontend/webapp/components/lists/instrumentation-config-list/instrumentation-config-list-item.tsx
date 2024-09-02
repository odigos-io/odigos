import React, { useState } from 'react';

import {
  HeaderItemWrapper,
  InstrumentationConfigHeaderContent,
  InstrumentationConfigItemContainer,
  InstrumentationConfigItemContent,
  InstrumentationConfigItemHeader,
  StyledItemCountContainer,
} from './style.styled';
import { KeyvalCheckbox, KeyvalTag, KeyvalText } from '@/design.system';
import InstrumentedLibraryOption from './library-option';
import { InstrumentationConfig, InstrumentationConfigLibrary } from '@/types';
import theme from '@/styles/palette';
import { ExpandIcon } from '@keyval-dev/design-system';
const InstrumentationConfigListItem = ({
  item,
  onOptionChange,
}: {
  children?: JSX.Element;
  item: InstrumentationConfig;
  onOptionChange: (item: any) => void;
}) => {
  const [isOpen, setIsOpen] = useState(false);

  const toggleLibOptions = () => {
    setIsOpen(!isOpen);
  };

  function renderOptions() {
    return item.instrumentationLibraries.map(
      (lib: InstrumentationConfigLibrary) => (
        <InstrumentedLibraryOption
          key={lib.instrumentationLibraryName}
          name={lib.instrumentationLibraryName}
          language={lib.language}
          selected={!!lib.selected}
          onChange={() => onLibChange(lib.instrumentationLibraryName)}
          disabled={false}
        />
      )
    );
  }

  function onOptionKeyChange() {
    const newLibraries = item.instrumentationLibraries.map((lib) => ({
      ...lib,
      selected: !item.optionValueBoolean,
    }));
    onOptionChange({
      ...item,
      optionValueBoolean: !item.optionValueBoolean,
      instrumentationLibraries: newLibraries,
    });
  }

  function onLibChange(name: string) {
    const instrumentationLibraries = item.instrumentationLibraries.map((lib) =>
      lib.instrumentationLibraryName === name
        ? { ...lib, selected: !lib.selected }
        : lib
    );

    const optionValueBoolean = instrumentationLibraries.some(
      (lib) => lib.selected
    );

    onOptionChange({
      ...item,
      optionValueBoolean,
      instrumentationLibraries,
    });
  }

  function getSelectedLibrariesCount() {
    return item.instrumentationLibraries?.filter(
      (lib: InstrumentationConfigLibrary) => lib.selected
    ).length;
  }

  function renderHeader() {
    return (
      <InstrumentationConfigItemHeader>
        <KeyvalCheckbox
          value={!!item.optionValueBoolean}
          onChange={onOptionKeyChange}
        />

        <InstrumentationConfigHeaderContent onClick={toggleLibOptions}>
          <div>
            <HeaderItemWrapper>
              <KeyvalText size={14} weight={600}>
                {item.optionKey}
              </KeyvalText>

              <KeyvalTag
                title={item.spanKind}
                color={
                  item.spanKind === 'Server'
                    ? theme.colors.dark_blue
                    : theme.colors.blue_grey
                }
              />
            </HeaderItemWrapper>
          </div>
          <HeaderItemWrapper>
            <StyledItemCountContainer>
              <KeyvalText
                color={theme.text.secondary}
                size={12}
              >{`${getSelectedLibrariesCount()}/${
                item.instrumentationLibraries?.length
              } libraries`}</KeyvalText>
            </StyledItemCountContainer>

            <ExpandIcon className={`dropdown-arrow ${isOpen && 'open'}`} />
          </HeaderItemWrapper>
        </InstrumentationConfigHeaderContent>
      </InstrumentationConfigItemHeader>
    );
  }

  return (
    <InstrumentationConfigItemContainer>
      {renderHeader()}
      <InstrumentationConfigItemContent open={isOpen}>
        {renderOptions()}
      </InstrumentationConfigItemContent>
    </InstrumentationConfigItemContainer>
  );
};

export default InstrumentationConfigListItem;

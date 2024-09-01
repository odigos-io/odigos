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
    return item.instrumentation_libraries.map(
      (lib: InstrumentationConfigLibrary) => (
        <InstrumentedLibraryOption
          key={lib.instrumentation_library_name}
          name={lib.instrumentation_library_name}
          language={lib.language}
          selected={!!lib.selected}
          onChange={() => onLibChange(lib.instrumentation_library_name)}
          disabled={!item.option_value_boolean}
        />
      )
    );
  }

  function onOptionKeyChange() {
    const newLibraries = item.instrumentation_libraries.map((lib) => ({
      ...lib,
      selected: !item.option_value_boolean,
    }));
    onOptionChange({
      ...item,
      option_value_boolean: !item.option_value_boolean,
      instrumentation_libraries: newLibraries,
    });
  }

  function onLibChange(name: string) {
    const instrumentation_libraries = item.instrumentation_libraries.map(
      (lib) =>
        lib.instrumentation_library_name === name
          ? { ...lib, selected: !lib.selected }
          : lib
    );

    const option_value_boolean = instrumentation_libraries.some(
      (lib) => lib.selected
    );

    onOptionChange({
      ...item,
      option_value_boolean,
      instrumentation_libraries,
    });
  }

  function getSelectedLibrariesCount() {
    return item.instrumentation_libraries?.filter(
      (lib: InstrumentationConfigLibrary) => lib.selected
    ).length;
  }

  function renderHeader() {
    return (
      <InstrumentationConfigItemHeader>
        <KeyvalCheckbox
          value={item.option_value_boolean}
          onChange={onOptionKeyChange}
        />

        <InstrumentationConfigHeaderContent onClick={toggleLibOptions}>
          <div>
            <HeaderItemWrapper>
              <KeyvalText size={14} weight={600}>
                {item.option_key}
              </KeyvalText>

              <KeyvalTag
                title={item.span_kind}
                color={
                  item.span_kind === 'Server'
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
                item.instrumentation_libraries.length
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

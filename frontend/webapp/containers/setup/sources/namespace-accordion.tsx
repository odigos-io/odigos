import { KeyvalCheckbox, KeyvalText } from '@/design.system';
import React, { useEffect, useState } from 'react';
import { WhiteArrowIcon } from '@keyval-dev/design-system';
import styled from 'styled-components';
interface AccordionItemData {
  title: string;
  items: any[];
}

interface AccordionProps {
  data: AccordionItemData[];
  setCurrentNamespace: (data: AccordionItemData) => void;
  onSelectItem: (item: any, ns: string) => void;
  onSelectAllChange: (ns, value) => void;
}

const ArrowIconWrapper = styled.div<{ expanded: boolean }>`
  margin-left: 24px;
  transform: rotate(${({ expanded }) => (expanded ? '90deg' : '-90deg')});
`;

export function NamespaceAccordion({
  data,
  setCurrentNamespace,
  onSelectItem,
  onSelectAllChange,
}: AccordionProps) {
  return (
    <div>
      {data.map((itemData, index) => (
        <AccordionItem
          key={index}
          data={itemData}
          onSelectItem={onSelectItem}
          setCurrentNamespace={setCurrentNamespace}
          onSelectAllChange={onSelectAllChange}
        />
      ))}
    </div>
  );
}

interface AccordionItemProps {
  data: any;
  setCurrentNamespace: (data: AccordionItemData) => void;
  onSelectItem: (item: any, ns: string) => void;
  onSelectAllChange: (ns, value) => void;
}

function AccordionItem({
  data,
  setCurrentNamespace,
  onSelectItem,
  onSelectAllChange,
}: AccordionItemProps) {
  const [isAllSelected, setIsAllSelected] = useState(false);
  const [expanded, setExpanded] = useState(false);

  useEffect(() => {
    const selectedItems = data.items?.filter((item) => item.selected);
    setIsAllSelected(
      selectedItems?.length === data?.items?.length && selectedItems?.length > 0
    );
  }, [data]);

  const handleSelectAllChange = (ns, value) => {
    onSelectAllChange(ns, value);
    setIsAllSelected(value);
  };

  const handleItemChange = (item) => {
    onSelectItem(item, data.title);
  };

  const handleExpand = () => {
    setCurrentNamespace(data);
    setExpanded(!expanded);
  };

  return (
    <div>
      <div
        style={{
          marginBottom: 8,
          display: 'flex',
          alignItems: 'center',
        }}
      >
        <KeyvalCheckbox
          value={isAllSelected}
          onChange={() => handleSelectAllChange(data.title, !isAllSelected)}
          label={''}
        />
        <div
          style={{
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
          }}
          onClick={handleExpand}
        >
          <KeyvalText style={{ marginLeft: 8, flex: 1, cursor: 'pointer' }}>
            {data.title}
          </KeyvalText>
          <ArrowIconWrapper expanded={expanded}>
            <WhiteArrowIcon size={10} />
          </ArrowIconWrapper>
        </div>
      </div>
      {expanded && (
        <div style={{ paddingLeft: '20px' }}>
          {data.items?.map((item, index) => (
            <div key={index} style={{ cursor: 'pointer', marginBottom: 8 }}>
              <KeyvalCheckbox
                value={item.selected}
                onChange={() => handleItemChange(item)}
                label={`${item.name} / ${item.kind.toLowerCase()}`}
              />
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

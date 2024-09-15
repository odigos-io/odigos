import React, { useState } from 'react';
import { KeyvalModal, KeyvalText, Table } from '@/design.system';
import { ManagedSource, Namespace } from '@/types';
import { SourcesTableRow } from './sources.table.row';
import { SourcesTableHeader } from './sources.table.header';
import { EmptyList } from '@/components/lists';
import { OVERVIEW } from '@/utils';

import { ModalPositionX, ModalPositionY } from '@/design.system/modal/types';

type TableProps = {
  data: ManagedSource[];
  namespaces?: Namespace[];
  onRowClick: (source: ManagedSource) => void;
  sortSources?: (condition: string) => void;
  filterSourcesByKind?: (kinds: string[]) => void;
  filterSourcesByNamespace?: (namespaces: string[]) => void;
  filterSourcesByLanguage?: (languages: string[]) => void;
  deleteSourcesHandler?: (sources: ManagedSource[]) => void;
  filterByConditionStatus?: (status: 'All' | 'True' | 'False') => void;
  filterByConditionMessage: (message: string[]) => void;
};

const SELECT_ALL_CHECKBOX = 'select_all';

export const ManagedSourcesTable: React.FC<TableProps> = ({
  data,
  namespaces,
  onRowClick,
  sortSources,
  filterSourcesByKind,
  deleteSourcesHandler,
  filterSourcesByNamespace,
  filterSourcesByLanguage,
  filterByConditionStatus,
  filterByConditionMessage,
}) => {
  const [selectedCheckbox, setSelectedCheckbox] = useState<string[]>([]);
  const [showModal, setShowModal] = useState(false);

  const modalConfig = {
    title: OVERVIEW.DELETE_SOURCE,
    showHeader: true,
    showOverlay: true,
    positionX: ModalPositionX.center,
    positionY: ModalPositionY.center,
    padding: '20px',
    footer: {
      primaryBtnText: OVERVIEW.CONFIRM_SOURCE_DELETE,
      primaryBtnAction: () => {
        setShowModal(false);
        parseSelectedSources();
      },
    },
  };
  const currentPageRef = React.useRef(1);

  function onPaginate(pageNumber: number) {
    currentPageRef.current = pageNumber;
  }

  function renderEmptyResult() {
    return <EmptyList title={OVERVIEW.EMPTY_SOURCE} />;
  }

  function onSelectedCheckboxChange(id: string) {
    const start = (currentPageRef.current - 1) * 10;
    const end = currentPageRef.current * 10;
    const slicedData = data.slice(start, end);
    if (id === SELECT_ALL_CHECKBOX) {
      if (selectedCheckbox.length === slicedData.length) {
        setSelectedCheckbox([]);
      } else {
        setSelectedCheckbox(slicedData.map((item) => JSON.stringify(item)));
      }
      return;
    }

    if (selectedCheckbox.includes(id)) {
      setSelectedCheckbox(selectedCheckbox.filter((item) => item !== id));
    } else {
      setSelectedCheckbox([...selectedCheckbox, id]);
    }
  }

  function parseSelectedSources() {
    const selectedSources = selectedCheckbox.map((item) => JSON.parse(item));
    deleteSourcesHandler && deleteSourcesHandler(selectedSources);
    setSelectedCheckbox([]);
  }

  function renderTableHeader() {
    return (
      <>
        <SourcesTableHeader
          data={data}
          namespaces={namespaces}
          sortSources={sortSources}
          filterSourcesByKind={filterSourcesByKind}
          filterSourcesByNamespace={filterSourcesByNamespace}
          filterSourcesByLanguage={filterSourcesByLanguage}
          selectedCheckbox={selectedCheckbox}
          onSelectedCheckboxChange={onSelectedCheckboxChange}
          deleteSourcesHandler={() => setShowModal(true)}
          filterByConditionStatus={filterByConditionStatus}
          filterByConditionMessage={filterByConditionMessage}
        />
        {showModal && (
          <KeyvalModal
            show={showModal}
            closeModal={() => setShowModal(false)}
            config={modalConfig}
          >
            <KeyvalText size={20} weight={600}>
              {`${OVERVIEW.DELETE} ${selectedCheckbox.length} sources`}
            </KeyvalText>
          </KeyvalModal>
        )}
      </>
    );
  }

  return (
    <>
      <Table<ManagedSource>
        data={data}
        renderTableHeader={renderTableHeader}
        onPaginate={onPaginate}
        renderEmptyResult={renderEmptyResult}
        renderTableRows={(item, index) => (
          <SourcesTableRow
            data={data}
            item={item}
            index={index}
            onRowClick={onRowClick}
            selectedCheckbox={selectedCheckbox}
            onSelectedCheckboxChange={onSelectedCheckboxChange}
          />
        )}
      />
    </>
  );
};

import React from "react";
import {
  SourcesListContainer,
  SourcesListWrapper,
  SourcesTitleWrapper,
  EmptyListWrapper,
} from "./sources.list.styled";
import { SourceCard } from "../source.card/source.card";
import { KeyvalLink, KeyvalText } from "@/design.system";
import { SETUP } from "@/utils/constants";
import Empty from "@/assets/images/empty-list.svg";

export function SourcesList({
  data,
  onItemClick,
  selectedData,
  onClearClick,
}: any) {
  function isFocus(currentCard: any) {
    const currentItem = selectedData?.objects?.filter(
      (item) => item.name === currentCard.name
    );

    return currentItem?.[0]?.selected || false;
  }

  function renderList() {
    return data?.map((item: any, index: number) => (
      <SourceCard
        key={index}
        item={item}
        onClick={() => onItemClick({ item, index })}
        focus={isFocus(item)}
      />
    ));
  }

  const isListEmpty = () => data?.length === 0;

  return !data ? null : (
    <SourcesListContainer>
      <SourcesTitleWrapper>
        <KeyvalText>{`${data?.length} ${SETUP.APPLICATIONS}`}</KeyvalText>
        <KeyvalLink onClick={onClearClick} value={SETUP.CLEAR_SELECTION} />
      </SourcesTitleWrapper>
      <SourcesListWrapper>
        {isListEmpty() ? (
          <EmptyListWrapper>
            <Empty />
          </EmptyListWrapper>
        ) : (
          renderList()
        )}
      </SourcesListWrapper>
    </SourcesListContainer>
  );
}

import React from "react";
import { SourcesListContainer } from "./sources.list.styled";
import { SourceCard } from "../source.card/source.card";

export function SourcesList({ data, onItemClick, selectedData }: any) {
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

  return <SourcesListContainer>{renderList()}</SourcesListContainer>;
}

import React from "react";
import {
  DestinationListWrapper,
  DestinationTypeTitleWrapper,
  EmptyListWrapper,
} from "./destination.list.styled";
import { KeyvalText } from "@/design.system";
import Empty from "@/assets/images/empty-list.svg";
import { DestinationCard } from "../destination.card/destination.card";

export function DestinationList({ data = [], onItemClick, selectedData }: any) {
  function renderList() {
    return data?.items?.map((item: any, index: number) => (
      <DestinationCard
        key={index}
        item={item}
        onClick={() => onItemClick({ item, index })}
        // focus={isFocus(item)}
      />
    ));
  }

  const isListEmpty = () => data?.length === 0;

  return (
    <>
      <DestinationTypeTitleWrapper>
        <KeyvalText>{`${data?.items?.length} ${data.name}`}</KeyvalText>
      </DestinationTypeTitleWrapper>
      <DestinationListWrapper>
        {isListEmpty() ? (
          <EmptyListWrapper>
            <Empty />
          </EmptyListWrapper>
        ) : (
          renderList()
        )}
      </DestinationListWrapper>
    </>
  );
}

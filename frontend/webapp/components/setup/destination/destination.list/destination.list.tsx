import React from "react";
import {
  DestinationListWrapper,
  DestinationTypeTitleWrapper,
} from "./destination.list.styled";
import { KeyvalText } from "@/design.system";
import { DestinationCard } from "../destination.card/destination.card";

export function DestinationList({ data = [], onItemClick, selectedData }: any) {
  function renderList() {
    return data?.items?.map((item: any, index: number) => (
      <DestinationCard
        key={index}
        item={item}
        onClick={() => onItemClick(item)}
        focus={selectedData?.type === item?.type}
      />
    ));
  }

  return (
    <>
      <DestinationTypeTitleWrapper>
        <KeyvalText>{`${data?.items?.length} ${data.name}`}</KeyvalText>
      </DestinationTypeTitleWrapper>
      <DestinationListWrapper>{renderList()}</DestinationListWrapper>
    </>
  );
}

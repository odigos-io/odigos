import React, { useMemo } from "react";
import {
  DestinationListWrapper,
  DestinationTypeTitleWrapper,
} from "./destination.list.styled";
import { KeyvalText } from "@/design.system";
import { DestinationCard } from "../destination.card/destination.card";

export function DestinationList({ data = [], onItemClick, sectionData }: any) {
  // const memorizedList = useMemo(() => {
  //   return data?.items?.map((item: any, index: number) => (
  //     <DestinationCard
  //       key={index}
  //       item={item}
  //       onClick={() => onItemClick(item)}
  //       focus={sectionData?.type === item?.type}
  //     />
  //   ));
  // }, [data, sectionData]);

  function renderList() {
    console.log("renderList");
    return data?.items?.map((item: any, index: number) => (
      <DestinationCard
        key={index}
        item={item}
        onClick={() => onItemClick(item)}
        focus={sectionData?.type === item?.type}
      />
    ));
  }

  return data?.items?.length ? (
    <>
      <DestinationTypeTitleWrapper>
        <KeyvalText>{`${data?.items?.length} ${data.name}`}</KeyvalText>
      </DestinationTypeTitleWrapper>
      <DestinationListWrapper>{renderList()}</DestinationListWrapper>
    </>
  ) : null;
}

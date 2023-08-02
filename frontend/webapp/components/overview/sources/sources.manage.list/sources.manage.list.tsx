import React from "react";
import {
  ManagedListWrapper,
  EmptyListWrapper,
  ManagedContainer,
} from "./sources.manage.styled";
import Empty from "@/assets/images/empty-list.svg";
import SourceManagedCard from "./sources.manage.card";
import { ManagedSource } from "@/types/sources";
import { KeyvalText } from "@/design.system";
import { OVERVIEW } from "@/utils/constants";

interface SourcesManagedListProps {
  data: ManagedSource[];
}

export function SourcesManagedList({ data = [] }: SourcesManagedListProps) {
  function renderSources() {
    return data.map((source: ManagedSource) => (
      <SourceManagedCard key={source?.name} item={source} />
    ));
  }

  return data.length === 0 ? (
    <EmptyListWrapper>
      <Empty />
    </EmptyListWrapper>
  ) : (
    <ManagedContainer>
      <KeyvalText>{`${data.length} ${OVERVIEW.MENU.SOURCES}`}</KeyvalText>
      <br />
      <ManagedListWrapper>{renderSources()}</ManagedListWrapper>
    </ManagedContainer>
  );
}

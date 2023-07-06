import React from "react";
import { SourcesListContainer } from "./sources.list.styled";
import { SourceCard } from "../source.card/source.card";

export function SourcesList({ data }: any) {
  function renderList() {
    return data?.map((item: any, index: number) => (
      <SourceCard key={index} item={item} />
    ));
  }

  return <SourcesListContainer>{renderList()}</SourcesListContainer>;
}

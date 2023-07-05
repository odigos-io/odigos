import React from "react";
import { SourcesListContainer } from "./sources.list.styled";
import { SourceCard } from "../source.card/source.card";

export function SourcesList() {
  function renderList() {
    return Array(10)
      .fill(0)
      .map((item, index) => <SourceCard />);
  }

  return <SourcesListContainer>{renderList()}</SourcesListContainer>;
}

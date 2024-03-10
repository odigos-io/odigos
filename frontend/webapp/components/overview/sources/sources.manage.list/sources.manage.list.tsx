import React from "react";
import { ManagedListWrapper, EmptyListWrapper } from "./sources.manage.styled";
import Empty from "@/assets/images/empty-list.svg";
import SourceManagedCard from "./sources.manage.card";

export function SourcesManagedList({ data = [1, 1, 1, 1] }) {
  function renderDestinations() {
    return data.map((source: any) => <SourceManagedCard />);
  }

  return (
    <>
      <ManagedListWrapper>
        {data?.length === 0 ? (
          <EmptyListWrapper>
            <Empty />
          </EmptyListWrapper>
        ) : (
          renderDestinations()
        )}
      </ManagedListWrapper>
    </>
  );
}

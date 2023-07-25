import React from "react";
import DestinationManagedCard from "./destination.managed.card";
import { styled } from "styled-components";

export const ManagedListWrapper = styled.div`
  width: 100%;
  display: flex;
  flex-wrap: wrap;
  gap: 24px;
  overflow-y: scroll;
  padding: 0px 36px;
`;

export default function DestinationsManagedList({ data }: any) {
  function renderDestinations() {
    return data.map((destination: any) => (
      <DestinationManagedCard key={destination?.name} item={destination} />
    ));
  }

  return <ManagedListWrapper>{renderDestinations()}</ManagedListWrapper>;
}

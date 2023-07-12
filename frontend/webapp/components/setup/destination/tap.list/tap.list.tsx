import { KeyvalTap } from "@/design.system";
import React from "react";
import styled from "styled-components";

const TapListWrapper = styled.div`
  display: flex;
`;

export function TapList({ list, gap = 8, tapStyle }: any) {
  function renderMonitoringOptions() {
    return list.map(({ icons, title, tapped }: any) => (
      <KeyvalTap tapped={tapped} icons={icons} title={title} style={tapStyle}>
        {tapped ? icons.focus() : icons.notFocus()}
      </KeyvalTap>
    ));
  }
  return (
    <TapListWrapper style={{ gap }}>{renderMonitoringOptions()}</TapListWrapper>
  );
}

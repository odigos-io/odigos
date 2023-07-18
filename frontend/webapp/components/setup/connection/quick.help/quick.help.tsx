import React from "react";
import { KeyvalText, KeyvalVideo } from "@/design.system";
import Note from "@/assets/icons/note.svg";
import { QuickHelpHeader, QuickHelpVideoWrapper } from "./quick.help.styled";
export function QuickHelp({ data }) {
  function renderVideoList() {
    return data?.map((video) => (
      <QuickHelpVideoWrapper key={video?.name}>
        <KeyvalVideo videoSrc={video?.src} title={video?.name} />
      </QuickHelpVideoWrapper>
    ));
  }

  return (
    <div>
      <QuickHelpHeader>
        <Note />
        <KeyvalText size={18} weight={600}>
          {"Quick help"}
        </KeyvalText>
      </QuickHelpHeader>
      {renderVideoList()}
    </div>
  );
}

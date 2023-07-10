import React, { useState } from "react";
import Question from "@/assets/icons/question.svg";
import { TooltipContentWrapper, TooltipWrapper } from "./tooltip.styled";
import { KeyvalText } from "../text/text";

export function KeyvalTooltip(props: any) {
  let timeout: ReturnType<typeof setTimeout>;
  const [active, setActive] = useState(false);

  const showTip = () => {
    timeout = setTimeout(() => {
      setActive(true);
    }, props.delay || 400);
  };

  const hideTip = () => {
    clearInterval(timeout);
    setActive(false);
  };

  return (
    <TooltipWrapper onMouseEnter={showTip} onMouseLeave={hideTip}>
      {active && (
        <TooltipContentWrapper>
          <KeyvalText size={12} weight={500}>
            {props.content}
          </KeyvalText>
        </TooltipContentWrapper>
      )}
      <Question />
    </TooltipWrapper>
  );
}

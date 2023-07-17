import React, { useState, MouseEvent } from "react";
import { styled } from "styled-components";
import Close from "@/assets/icons/close.svg";
import { KeyvalText } from "../text/text";
type VideoComponentProps = {
  videoSrc: string;
};

const StyledVideo = styled.video`
  width: 100%;
  max-width: 600px;
  border-radius: 8px;
  margin-top: 8px;
`;

const StyledLargeVideo = styled.video`
  width: 980px;
  border-radius: 8px;
`;

const LargeVideoHeader = styled.div`
  width: 980px;

  display: flex;
  justify-content: space-between;
  margin-bottom: 21px;
`;

const LargeVideoContainer = styled.div`
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  background: rgba(0, 0, 0, 0.65);
  z-index: 9999;
`;

export function KeyvalVideo({ videoSrc }: VideoComponentProps) {
  const [isLarge, setIsLarge] = useState(false);

  const handleClick = (): void => {
    setIsLarge(true);
  };

  const handleClose = (): void => {
    setIsLarge(false);
  };

  return (
    <div>
      {isLarge ? (
        <LargeVideoContainer>
          <LargeVideoHeader>
            <KeyvalText size={20} weight={600}>
              {"DataDog: Find API Key"}
            </KeyvalText>
            <Close onClick={handleClose} style={{ cursor: "pointer" }} />
          </LargeVideoHeader>
          <StyledLargeVideo src={videoSrc} controls autoPlay />
        </LargeVideoContainer>
      ) : (
        <>
          <KeyvalText size={16} weight={600}>
            {"DataDog: Find API Key"}
          </KeyvalText>
          <StyledVideo src={videoSrc} onClick={handleClick} />
        </>
      )}
    </div>
  );
}

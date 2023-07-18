import React, { useState } from "react";
import CloseIcon from "@/assets/icons/close.svg";
import PlayerIcon from "@/assets/icons/player.svg";
import { KeyvalText } from "../text/text";
import {
  ImagePreviewWrapper,
  PlayerIconWrapper,
  LargePlayerIconWrapper,
  StyledLargeVideo,
  LargeVideoHeader,
  LargeVideoContainer,
} from "./video.styled";

type VideoComponentProps = {
  videoSrc: string;
  title?: string;
};

export function KeyvalVideo({ videoSrc, title }: VideoComponentProps) {
  const [isLarge, setIsLarge] = useState(false);
  const [pause, setPause] = useState(true);

  const handleClick = (): void => {
    setIsLarge(true);
  };

  const handleClose = (): void => {
    setIsLarge(false);
    setPause(true);
  };

  const renderSmallView = (): JSX.Element => (
    <>
      <KeyvalText size={16} weight={600}>
        {title}
      </KeyvalText>
      <ImagePreviewWrapper
        onClick={handleClick}
        url={
          "https://a.fsdn.com/con/app/proj/chronosphere.s/screenshots/Captura%20de%20Pantalla%202021-01-22%20a%20la%28s%29%2016.27.35.png/max/max/1"
        }
      >
        <PlayerIconWrapper>
          <PlayerIcon width={30} />
        </PlayerIconWrapper>
      </ImagePreviewWrapper>
    </>
  );

  const renderLargeView = (): JSX.Element => (
    <LargeVideoContainer>
      <LargeVideoHeader>
        <KeyvalText size={20} weight={600}>
          {title}
        </KeyvalText>
        <CloseIcon onClick={handleClose} style={{ cursor: "pointer" }} />
      </LargeVideoHeader>
      {!pause ? (
        <StyledLargeVideo src={videoSrc} autoPlay controls />
      ) : (
        <ImagePreviewWrapper
          url={
            "https://a.fsdn.com/con/app/proj/chronosphere.s/screenshots/Captura%20de%20Pantalla%202021-01-22%20a%20la%28s%29%2016.27.35.png/max/max/1"
          }
          style={{ width: 980, height: 560 }}
          onClick={() => setPause(false)}
        >
          <LargePlayerIconWrapper>
            <PlayerIcon width={80} />
          </LargePlayerIconWrapper>
        </ImagePreviewWrapper>
      )}
    </LargeVideoContainer>
  );

  return <div>{isLarge ? renderLargeView() : renderSmallView()}</div>;
}

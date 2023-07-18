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
          "https://4260490853-files.gitbook.io/~/files/v0/b/gitbook-x-prod.appspot.com/o/spaces%2F-M1bR_-Od0islbiOl4G0%2Fuploads%2Fgit-blob-80d65344037e059578979909f052cd2c5f09163d%2Fdatadog.png?alt=media"
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
            "https://4260490853-files.gitbook.io/~/files/v0/b/gitbook-x-prod.appspot.com/o/spaces%2F-M1bR_-Od0islbiOl4G0%2Fuploads%2Fgit-blob-80d65344037e059578979909f052cd2c5f09163d%2Fdatadog.png?alt=media"
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

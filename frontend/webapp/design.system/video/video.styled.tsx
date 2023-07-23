import { styled } from "styled-components";

interface ImagePreviewWrapperProps {
  url: string | undefined;
}

export const ImagePreviewWrapper = styled.div<ImagePreviewWrapperProps>`
  position: relative;
  margin-top: 8px;
  border-radius: 8px;
  width: 240px;
  height: 140px;
  cursor: pointer;
  background: ${({ url }) => `linear-gradient(
      0deg,
      rgba(2, 20, 30, 0.2) 0%,
      rgba(2, 20, 30, 0.2) 100%
    ),
    url(${url}),
    lightgray 50%`};
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
`;

export const PlayerIconWrapper = styled.div`
  position: absolute;
  margin-left: auto;
  margin-right: auto;
  left: 0;
  right: 0;
  top: 30px;
  text-align: center;
`;
export const LargePlayerIconWrapper = styled(PlayerIconWrapper)`
  top: 40%;
`;

export const StyledLargeVideo = styled.video`
  width: 980px;
  border-radius: 8px;
`;

export const LargeVideoHeader = styled.div`
  width: 980px;

  display: flex;
  justify-content: space-between;
  margin-bottom: 21px;
`;

export const LargeVideoContainer = styled.div`
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

import React from "react";
import { styled } from "styled-components";

interface KeyvalImageProps {
  src: string;
  alt?: string;
  width?: number;
  height?: number;
}

const StyledImage = styled.img`
  border-radius: 10px;
`;

export function KeyvalImage({
  src,
  alt,
  width = 200,
  height = 200,
}: KeyvalImageProps) {
  return <StyledImage src={src} alt={alt} width={width} height={height} />;
}

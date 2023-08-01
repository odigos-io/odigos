import Image from "next/image";
import React from "react";

interface KeyvalImageProps {
  src: string;
  alt?: string;
  width?: number;
  height?: number;
  style?: React.CSSProperties;
}

const IMAGE_STYLE: React.CSSProperties = {
  borderRadius: 10,
};

export function KeyvalImage({
  src,
  alt,
  width = 56,
  height = 56,
  style = {},
}: KeyvalImageProps) {
  return (
    <Image
      src={src}
      alt={alt || ""}
      width={width}
      height={height}
      style={{ ...IMAGE_STYLE, ...style }}
    />
  );
}

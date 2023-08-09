import React from "react";
import { Video } from "@keyval-dev/design-system";

type VideoComponentProps = {
  videoSrc: string;
  title?: string;
  thumbnail?: string | undefined;
};

export function KeyvalVideo(props: VideoComponentProps) {
  return <Video {...props} />;
}

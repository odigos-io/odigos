import React from "react";
import { Video } from "@odigos-io/design-system";

type VideoComponentProps = {
  videoSrc: string;
  title?: string;
  thumbnail?: string | undefined;
};

export function KeyvalVideo(props: VideoComponentProps) {
  return <Video {...props} />;
}

import React from "react";
import Lottie from "react-lottie";

export function KeyvalLottie({
  loop,
  autoplay,
  animationData,
  width = 100,
  height = 100,
}) {
  const defaultOptions = {
    loop,
    autoplay,
    animationData,
    rendererSettings: {
      preserveAspectRatio: "xMidYMid slice",
    },
  };

  return <Lottie options={defaultOptions} height={height} width={width} />;
}

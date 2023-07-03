import React from "react";

type TextProps = {
  value: string;
};

export default function Text({ value }: TextProps) {
  return <span>{value}</span>;
}

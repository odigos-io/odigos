import React from "react";
import { Tag } from "@keyval-dev/design-system";

interface TagProps {
  title: string;
  color?: string;
}

export function KeyvalTag({ title = "", color = "#033869" }: TagProps) {
  return <Tag color={color} title={title} />;
}

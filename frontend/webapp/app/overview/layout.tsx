"use client";
import { Menu } from "@/components/side.menu";
import React from "react";

const LAYOUT_STYLE = {
  width: "100%",
  height: "100%",
  display: "flex",
};

const CHILDREN_STYLE = {
  width: "100%",
  height: "100%",
};

export default function Layout({ children }: { children: React.ReactNode }) {
  return (
    <div style={LAYOUT_STYLE}>
      <Menu />
      <div style={CHILDREN_STYLE}>{children}</div>
    </div>
  );
}

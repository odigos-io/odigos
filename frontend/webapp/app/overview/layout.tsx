"use client";
import { Menu } from "@/components/side.menu";
import React from "react";

export default function Layout({ children }: { children: React.ReactNode }) {
  return (
    <body>
      <div style={{ width: "100%", height: "100%", display: "flex" }}>
        <Menu />
        <div style={{ width: "100%", height: "100%" }}>{children}</div>
      </div>
    </body>
  );
}

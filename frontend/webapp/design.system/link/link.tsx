import React from "react";
import { Link } from "@keyval-dev/design-system";

interface KeyvalLinkProps {
  value: string;
  onClick?: () => void;
}

export function KeyvalLink(props: KeyvalLinkProps) {
  return <Link {...props} />;
}

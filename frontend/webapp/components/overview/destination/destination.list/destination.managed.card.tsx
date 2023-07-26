import React, { useMemo } from "react";
import { KeyvalImage, KeyvalText } from "@/design.system";
import { MONITORING_OPTIONS } from "@/components/setup/destination/utils";
import { TapList } from "@/components/setup/destination/tap.list/tap.list";
import {
  CardWrapper,
  Border,
  ManagedWrapper,
  ApplicationNameWrapper,
} from "./destination.list.styled";
import theme from "@/styles/palette";
import { OVERVIEW } from "@/utils/constants";

const TEXT_STYLE: React.CSSProperties = {
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
  overflow: "hidden",
};
const LOGO_STYLE: React.CSSProperties = {
  padding: 4,
  backgroundColor: theme.colors.white,
};
const TAP_STYLE: React.CSSProperties = { padding: "4px 8px", gap: 4 };

export default function DestinationManagedCard({
  onClick,
  item: {
    destination_type: { image_url, display_name, supported_signals },
    name,
    signals,
  },
}) {
  const monitors = useMemo(() => {
    return Object?.entries(supported_signals).reduce((acc, [key, _]) => {
      const monitor = MONITORING_OPTIONS.find(
        (option) => option.title.toLowerCase() === key
      );
      if (monitor && supported_signals[key].supported) {
        return [...acc, { ...monitor, tapped: signals[key] }];
      }

      return acc;
    }, []);
  }, [JSON.stringify(supported_signals)]);

  return (
    <CardWrapper onClick={onClick}>
      <KeyvalImage src={image_url} width={56} height={56} style={LOGO_STYLE} />
      <ApplicationNameWrapper>
        <KeyvalText size={20} weight={700} style={TEXT_STYLE}>
          {display_name}
        </KeyvalText>
        <KeyvalText size={20} style={TEXT_STYLE}>
          {name}
        </KeyvalText>
      </ApplicationNameWrapper>
      <TapList gap={4} list={monitors} tapStyle={TAP_STYLE} />
      {/* <Border />
      <ManagedWrapper onClick={onClick}>
        <KeyvalText>{OVERVIEW.MANAGE}</KeyvalText>
      </ManagedWrapper> */}
    </CardWrapper>
  );
}

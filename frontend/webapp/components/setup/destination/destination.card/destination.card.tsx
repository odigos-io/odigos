import { KeyvalCard, KeyvalImage, KeyvalText } from "@/design.system";
import React, { useMemo } from "react";
import {
  ApplicationNameWrapper,
  DestinationCardWrapper,
} from "./destination.card.styled";
import { TapList } from "../tap.list/tap.list";
import { MONITORING_OPTIONS } from "../utils";

const TEXT_STYLE = {
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
  overflow: "hidden",
};

export function DestinationCard({ item, onClick, focus }: any) {
  const monitors = useMemo(() => {
    const data = MONITORING_OPTIONS.map((monitor: any) => {
      const isSupported =
        item?.supported_signals?.[monitor.title.toLowerCase()]?.supported;

      if (isSupported) {
        return {
          ...monitor,
          tapped: true,
        };
      }
    });

    return data.filter(Boolean);
  }, [JSON.stringify(item)]);

  return (
    <KeyvalCard focus={focus}>
      <DestinationCardWrapper onClick={onClick}>
        <KeyvalImage src={item?.image_url} width={56} height={56} />
        <ApplicationNameWrapper>
          <KeyvalText size={20} weight={700} style={TEXT_STYLE}>
            {item.display_type}
          </KeyvalText>
        </ApplicationNameWrapper>
        <TapList list={monitors} />
      </DestinationCardWrapper>
    </KeyvalCard>
  );
}

import React, { useMemo } from "react";
import { KeyvalImage, KeyvalText } from "@/design.system";
import { styled } from "styled-components";
import { MONITORING_OPTIONS } from "@/components/setup/destination/utils";
import { TapList } from "@/components/setup/destination/tap.list/tap.list";

const TEXT_STYLE: React.CSSProperties = {
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
  overflow: "hidden",
};
const LOGO_STYLE: React.CSSProperties = { padding: 4, backgroundColor: "#fff" };
const TAP_STYLE: React.CSSProperties = { padding: "4px 8px", gap: 4 };

type DestinationCardContentProps = {
  // item: Destination;
  onClick: () => void;
};

const CardWrapper = styled.div`
  display: flex;
  width: 366px;
  padding-top: 32px;
  padding-bottom: 24px;
  flex-direction: column;
  align-items: center;
  border-radius: 24px;
  border: 1px solid var(--dark-mode-dark-3, #203548);
  background: var(--dark-mode-dark-1, #07111a);
  display: flex;
  align-items: center;
  flex-direction: column;
`;

const Border = styled.div`
  width: 368px;
  height: 1px;
  margin: 24px 0;
  background: var(--dark-mode-dark-3, #203548);
`;

const ManagedWrapper = styled.div`
  display: flex;
  padding: 8px 12px;
  align-items: flex-start;
  border-radius: 10px;
  border: 1px solid var(--dark-mode-odigos-torquiz, #96f2ff);
  cursor: pointer;
`;

export const ApplicationNameWrapper = styled.div`
  display: inline-block;
  text-overflow: ellipsis;
  max-width: 224px;
  text-align: center;
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 12px;
  margin-bottom: 20px;
`;

export default function DestinationManagedCard({
  item: {
    destination_type: { image_url, display_name, supported_signals },
    name,
    signals,
  },
}) {
  const monitors = useMemo(() => {
    return Object?.entries(supported_signals).reduce((acc, [key, value]) => {
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
    <CardWrapper>
      <KeyvalImage src={image_url} width={56} height={56} style={LOGO_STYLE} />
      <ApplicationNameWrapper>
        <KeyvalText size={20} weight={700} style={TEXT_STYLE}>
          {display_name}
        </KeyvalText>
        {name && (
          <KeyvalText size={20} style={TEXT_STYLE}>
            {name}
          </KeyvalText>
        )}
      </ApplicationNameWrapper>
      <TapList gap={4} list={monitors} tapStyle={TAP_STYLE} />
      <Border />
      <ManagedWrapper>
        <KeyvalText>{"Managed"}</KeyvalText>
      </ManagedWrapper>
    </CardWrapper>
  );
}

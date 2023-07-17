import { KeyvalCheckbox, KeyvalInput, KeyvalText } from "@/design.system";
import React, { useState, useEffect } from "react";
import {
  CheckboxWrapper,
  ConnectionMonitorsWrapper,
  DestinationNameWrapper,
} from "./create.connection.form.styled";

const MONITORS = [
  { id: "logs", label: "Logs", checked: true },
  { id: "metrics", label: "Metrics", checked: true },
  { id: "traces", label: "Traces", checked: true },
];

export function CreateConnectionForm() {
  const [destinationName, setDestinationName] = useState<string>("");
  const [selectedMonitors, setSelectedMonitors] = useState(MONITORS);

  const handleCheckboxChange = (id: string) => {
    setSelectedMonitors((prevCheckboxes) =>
      prevCheckboxes.map((checkbox) =>
        checkbox.id === id
          ? { ...checkbox, checked: !checkbox.checked }
          : checkbox
      )
    );
  };

  return (
    <>
      <KeyvalText size={18} weight={600}>
        {"Create connection"}
      </KeyvalText>
      <ConnectionMonitorsWrapper>
        <KeyvalText size={14}>{"This connection will monitor:"}</KeyvalText>
        <CheckboxWrapper>
          {selectedMonitors.map((checkbox) => (
            <KeyvalCheckbox
              value={checkbox?.checked}
              onChange={() => handleCheckboxChange(checkbox?.id)}
              label={checkbox?.label}
            />
          ))}

          {/* <KeyvalCheckbox value={true} onChange={() => {}} label="Metrics" />
          <KeyvalCheckbox value={true} onChange={() => {}} label="Traces" /> */}
        </CheckboxWrapper>
      </ConnectionMonitorsWrapper>
      <DestinationNameWrapper>
        <KeyvalInput
          label={"Destination Name"}
          value={destinationName}
          onChange={setDestinationName}
        />
      </DestinationNameWrapper>
    </>
  );
}

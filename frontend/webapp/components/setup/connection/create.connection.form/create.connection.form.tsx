import React, { useState, useEffect } from "react";
import {
  DangerZone,
  KeyvalButton,
  KeyvalCheckbox,
  KeyvalImage,
  KeyvalInput,
  KeyvalModal,
  KeyvalText,
} from "@/design.system";
import {
  CheckboxWrapper,
  ConnectionMonitorsWrapper,
  DynamicFieldsWrapper,
  FieldWrapper,
  CreateDestinationButtonWrapper,
} from "./create.connection.form.styled";
import { renderFields } from "./dynamic.fields";
import { SETUP } from "@/utils/constants";
import { DestinationBody } from "@/containers/setup/connection/connection.section";
import { Field } from "@/types/destinations";
import { ModalPositionX, ModalPositionY } from "@/design.system/modal/types";
import { ConnectionsIcons } from "../connections_icons/connections_icons";
import theme from "@/styles/palette";
import Connect from "assets/icons/connect.svg";
interface CreateConnectionFormProps {
  fields: Field[];
  image?: any;
  onSubmit: (formData: DestinationBody) => void;
  supportedSignals: {
    [key: string]: {
      supported: boolean;
    };
  };
  dynamicFieldsValues?: {
    [key: string]: any;
  };
  destinationNameValue?: string | null;
  checkboxValues?: {
    [key: string]: boolean;
  };
}

const MONITORS = [
  { id: "logs", label: SETUP.MONITORS.LOGS, checked: true },
  { id: "metrics", label: SETUP.MONITORS.METRICS, checked: true },
  { id: "traces", label: SETUP.MONITORS.TRACES, checked: true },
];

export function CreateConnectionForm({
  fields,
  onSubmit,
  supportedSignals,
  dynamicFieldsValues,
  destinationNameValue,
  checkboxValues,
  image,
}: CreateConnectionFormProps) {
  const [destinationName, setDestinationName] = useState<string>(
    destinationNameValue || ""
  );
  const [selectedMonitors, setSelectedMonitors] = useState(MONITORS);
  const [dynamicFields, setDynamicFields] = useState(dynamicFieldsValues || {});
  const [isCreateButtonDisabled, setIsCreateButtonDisabled] = useState(true);
  const [showModal, setShowModal] = useState(false);

  useEffect(() => {
    isFormValid();
  }, [destinationName, dynamicFields]);

  useEffect(() => {
    filterSupportedMonitors();
  }, [supportedSignals]);

  function filterSupportedMonitors() {
    const data: any = !checkboxValues
      ? MONITORS
      : MONITORS.map((monitor) => ({
          ...monitor,
          checked: checkboxValues[monitor.id],
        }));

    setSelectedMonitors(
      data.filter(({ id }) => supportedSignals[id]?.supported)
    );
  }

  const handleCheckboxChange = (id: string) => {
    setSelectedMonitors((prevSelectedMonitors) => {
      const totalSelected = prevSelectedMonitors.filter(
        ({ checked }) => checked
      ).length;

      if (
        totalSelected === 1 &&
        prevSelectedMonitors.find((item) => item.id === id)?.checked
      ) {
        return prevSelectedMonitors; // Prevent unchecking the last selected checkbox
      }

      const updatedMonitors = prevSelectedMonitors.map((checkbox) =>
        checkbox.id === id
          ? { ...checkbox, checked: !checkbox.checked }
          : checkbox
      );

      return updatedMonitors;
    });
  };

  function handleDynamicFieldChange(name: string, value: string) {
    setDynamicFields((prevFields) => ({ ...prevFields, [name]: value }));
  }

  function isFormValid() {
    const dynamicFieldsValues = Object.values(dynamicFields);
    const isValid =
      !!destinationName &&
      dynamicFieldsValues.every((field: Field) => field) &&
      dynamicFieldsValues.length === fields?.length;

    setIsCreateButtonDisabled(!isValid);
  }

  function onCreateClick() {
    const signals = selectedMonitors.reduce(
      (acc, { id, checked }) => ({ ...acc, [id]: checked }),
      {}
    );
    const body = {
      name: destinationName,
      signals,
      fields: dynamicFields,
    };
    onSubmit(body);
  }

  const modal = {
    title: `Delete destination`,
    showHeader: true,
    showOverlay: true,
    positionX: ModalPositionX.center,
    positionY: ModalPositionY.center,
    padding: "20px",
    footer: {
      primaryBtnText: "I want to delete this destination",
      primaryBtnAction: () => setShowModal(false),
    },
  };

  return (
    <div>
      <KeyvalText size={18} weight={600}>
        {SETUP.CREATE_CONNECTION}
      </KeyvalText>
      <ConnectionMonitorsWrapper>
        <KeyvalText size={14}>{SETUP.CONNECTION_MONITORS}</KeyvalText>
        <CheckboxWrapper>
          {selectedMonitors.map((checkbox) => (
            <KeyvalCheckbox
              key={checkbox?.id}
              value={checkbox?.checked}
              onChange={() => handleCheckboxChange(checkbox?.id)}
              label={checkbox?.label}
            />
          ))}
        </CheckboxWrapper>
      </ConnectionMonitorsWrapper>
      <FieldWrapper>
        <KeyvalInput
          style={{ height: 36 }}
          label={SETUP.DESTINATION_NAME}
          value={destinationName}
          onChange={setDestinationName}
        />
      </FieldWrapper>
      <DynamicFieldsWrapper>
        {renderFields(fields, dynamicFields, handleDynamicFieldChange)}
      </DynamicFieldsWrapper>
      <CreateDestinationButtonWrapper>
        <KeyvalButton disabled={isCreateButtonDisabled} onClick={onCreateClick}>
          <KeyvalText color={"#203548"} size={14} weight={600}>
            {dynamicFieldsValues
              ? SETUP.UPDATE_DESTINATION
              : SETUP.CREATE_DESTINATION}
          </KeyvalText>
        </KeyvalButton>
      </CreateDestinationButtonWrapper>
      {dynamicFieldsValues && (
        <>
          <FieldWrapper>
            <DangerZone
              title="Delete this destination"
              subTitle="This action cannot be undone. This will permanently delete the destination and all associated data."
              btnText="Delete"
              onClick={() => setShowModal(true)}
            />
          </FieldWrapper>
          <KeyvalModal
            show={showModal}
            setShow={() => setShowModal(false)}
            config={modal}
          >
            <br />
            <ConnectionsIcons
              icon={image}
              imageStyle={{ border: "solid 1px #ededed" }}
            />
            <br />
            <KeyvalText color={theme.text.primary} size={20} weight={600}>
              {`Delete ${destinationName}`}
            </KeyvalText>
          </KeyvalModal>
        </>
      )}
    </div>
  );
}

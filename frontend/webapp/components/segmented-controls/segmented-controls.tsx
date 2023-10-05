"use client";
import React from "react";
import styled from "styled-components";

const SegmentedControlsWrapper = styled.div`
  display: inline-flex;
  justify-content: space-between;
  border-radius: 10px;
  max-width: 500px;
  padding: 12px;
  margin: auto;
  overflow: hidden;
  position: relative;
`;
const SegmentedControlsOption = styled.div`
  color: ${({ theme }) => theme.colors.white};
  padding: 8px 12px;
  position: relative;
  text-align: center;
  z-index: 1;
  border: ${({ theme }) => `1px solid  ${theme.colors.secondary}`};
  filter: brightness(70%);
  &.active {
    filter: brightness(100%);
  }
  &:first-child {
    border-top-left-radius: 10px;
    border-bottom-left-radius: 10px;
    padding-left: 16px;
  }
  &:last-child {
    border-top-right-radius: 10px;
    border-bottom-right-radius: 10px;
    padding-right: 16px;
  }
`;

const SegmentedControlsInput = styled.input`
  opacity: 0;
  margin: 0;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  position: absolute;
  width: 100%;
  cursor: pointer;
  height: 100%;
`;

export function SegmentedControls({
  options,
  selected,
  onChange,
}: {
  options: string[];
  selected: string;
  onChange: (selected: string) => void;
}) {
  return (
    <SegmentedControlsWrapper>
      {options?.map((option) => (
        <SegmentedControlsOption
          key={option}
          className={`${option === selected ? "active" : ""}`}
        >
          <SegmentedControlsInput
            type="radio"
            value={option}
            name={option}
            onChange={() => onChange(option)}
            checked={option === selected}
          />
          <label htmlFor={option}>{option}</label>
        </SegmentedControlsOption>
      ))}
    </SegmentedControlsWrapper>
  );
}

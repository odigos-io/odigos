import React, { useState } from "react";
import Open from "@/assets/icons/expand-arrow.svg";
import { DropdownHeader, DropdownWrapper } from "./drop.down.styled";
import { KeyvalText } from "../text/text";

interface DropDownItem {
  id: number;
  label: string;
}
interface KeyvalDropDownProps {
  data: DropDownItem[];
}

export function KeyvalDropDown({ data }: KeyvalDropDownProps) {
  const [isOpen, setOpen] = useState(false);
  const [selectedItem, setSelectedItem] = useState(null);

  const toggleDropdown = () => setOpen(!isOpen);

  const handleItemClick = (item: any) => {
    setSelectedItem(item.id);
    setOpen(false);
  };

  return (
    <div style={{ height: 37 }}>
      <DropdownWrapper>
        <DropdownHeader onClick={toggleDropdown}>
          {selectedItem
            ? data?.find((item) => item?.id == selectedItem)?.label
            : "Select your destination"}
          <Open className={`dropdown-arrow ${isOpen && "open"}`} />
        </DropdownHeader>
        <div className={`dropdown-body ${isOpen && "open"}`}>
          {data.map((item) => (
            <div
              className="dropdown-item"
              onClick={(e: any) => handleItemClick(item)}
            >
              <KeyvalText>{item.label}</KeyvalText>
            </div>
          ))}
        </div>
      </DropdownWrapper>
    </div>
  );
}

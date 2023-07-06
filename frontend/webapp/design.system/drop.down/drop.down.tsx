import React, { useEffect, useState } from "react";
import Open from "@/assets/icons/expand-arrow.svg";
import { DropdownHeader, DropdownWrapper } from "./drop.down.styled";
import { KeyvalText } from "../text/text";

interface DropDownItem {
  id: number;
  label: string;
}
interface KeyvalDropDownProps {
  data: DropDownItem[];
  onChange: (item: DropDownItem) => void;
}

const SELECTED_ITEM = "Select item";

export function KeyvalDropDown({ data, onChange }: KeyvalDropDownProps) {
  const [isOpen, setOpen] = useState(false);
  const [selectedItem, setSelectedItem] = useState<any>(data[0] || null);

  const toggleDropdown = () => setOpen(!isOpen);

  const handleItemClick = (item: DropDownItem) => {
    onChange(item);
    setSelectedItem(item);
    setOpen(false);
  };

  return (
    <div style={{ height: 37 }}>
      <DropdownWrapper>
        <DropdownHeader onClick={toggleDropdown}>
          {selectedItem ? selectedItem.label : SELECTED_ITEM}
          <Open className={`dropdown-arrow ${isOpen && "open"}`} />
        </DropdownHeader>
        <div className={`dropdown-body ${isOpen && "open"}`}>
          {data.map((item) => (
            <div
              key={item.id}
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

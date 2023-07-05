import React, { useState } from "react";
import "./drop.down.css";

interface KeyvalDropDownProps {}

const data = [
  { id: 0, label: "Istanbul, TR (AHL)" },
  { id: 1, label: "Paris, FR (CDG)" },
];

export function KeyvalDropDown({}: KeyvalDropDownProps) {
  const [isOpen, setOpen] = useState(false);
  const [items, setItem] = useState(data);
  const [selectedItem, setSelectedItem] = useState(null);

  const toggleDropdown = () => setOpen(!isOpen);

  const handleItemClick = (id: any) => {
    console.log({ id });
    selectedItem == id ? setSelectedItem(null) : setSelectedItem(id);
  };

  return (
    <div className="dropdown">
      <div className="dropdown-header" onClick={toggleDropdown}>
        {selectedItem
          ? items?.find((item) => item?.id == selectedItem)?.label
          : "Select your destination"}
        <i className={`fa fa-chevron-right icon ${isOpen && "open"}`}></i>
      </div>
      <div className={`dropdown-body ${isOpen && "open"}`}>
        {items.map((item) => (
          <div
            className="dropdown-item"
            onClick={(e: any) => handleItemClick(e.target?.id)}
            // id={item?.id || 1}
          >
            <span
              className={`dropdown-item-dot ${
                item.id == selectedItem && "selected"
              }`}
            >
              •{" "}
            </span>
            {item.label}
          </div>
        ))}
      </div>
    </div>
  );
}

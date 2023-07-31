export enum ModalPositionX {
  center = "center",
  right = "right",
  left = "left",
}

export enum ModalPositionY {
  center = "center",
  start = "start",
  end = "end",
}

export interface ModalConfig {
  title: string;
  showHeader: boolean;
  positionX: ModalPositionX;
  positionY: ModalPositionY;
  padding: string;
  showOverlay: boolean;
  footer?: {
    primaryBtnText: string;
    secondaryBtnText?: string;
    primaryBtnAction: () => void;
    secondaryBtnAction?: () => void;
  };
}

export interface Props {
  show: boolean;
  config: ModalConfig;
  closeModal: () => void;
  children: JSX.Element | JSX.Element[];
}

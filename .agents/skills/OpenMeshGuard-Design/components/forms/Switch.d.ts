import * as React from 'react';

export interface SwitchProps {
  checked?: boolean;
  /** Receives the next boolean value. */
  onChange?: (checked: boolean) => void;
  label?: React.ReactNode;
  disabled?: boolean;
  id?: string;
  style?: React.CSSProperties;
}

/** On/off toggle switch. */
export function Switch(props: SwitchProps): JSX.Element;

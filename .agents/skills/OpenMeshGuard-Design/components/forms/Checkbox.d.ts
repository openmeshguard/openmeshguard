import * as React from 'react';

export interface CheckboxProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'onChange'> {
  checked?: boolean;
  onChange?: React.ChangeEventHandler<HTMLInputElement>;
  label?: React.ReactNode;
  disabled?: boolean;
}

/** Checkbox with inline label. */
export function Checkbox(props: CheckboxProps): JSX.Element;

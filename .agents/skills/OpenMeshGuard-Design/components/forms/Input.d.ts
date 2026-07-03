import * as React from 'react';

export interface InputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'size'> {
  label?: string;
  hint?: string;
  /** Error message; also turns the field red. */
  error?: string;
  leadingIcon?: React.ReactNode;
  size?: 'sm' | 'md';
  containerStyle?: React.CSSProperties;
}

/**
 * Text input with label, hint, and error states.
 * @startingPoint section="Forms" subtitle="Inputs, selects, checkbox, switch" viewport="700x260"
 */
export function Input(props: InputProps): JSX.Element;

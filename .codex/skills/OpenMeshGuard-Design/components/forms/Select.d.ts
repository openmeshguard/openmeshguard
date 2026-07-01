import * as React from 'react';

export interface SelectOption { value: string; label: string; }

export interface SelectProps extends Omit<React.SelectHTMLAttributes<HTMLSelectElement>, 'size'> {
  label?: string;
  hint?: string;
  /** Options list; alternatively pass <option> children. */
  options?: SelectOption[];
  size?: 'sm' | 'md';
  containerStyle?: React.CSSProperties;
}

/** Styled native select. */
export function Select(props: SelectProps): JSX.Element;

import * as React from 'react';

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  /** Visual style. Use `primary` for the main action, `danger` for destructive. */
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
  size?: 'sm' | 'md' | 'lg';
  /** Icon element rendered before the label. */
  leftIcon?: React.ReactNode;
  /** Icon element rendered after the label. */
  rightIcon?: React.ReactNode;
  fullWidth?: boolean;
  children?: React.ReactNode;
}

/**
 * Primary interactive button for OpenMeshGuard. Sentence-case labels only.
 * @startingPoint section="Actions" subtitle="Buttons — primary, secondary, ghost, danger" viewport="700x150"
 */
export function Button(props: ButtonProps): JSX.Element;

import * as React from 'react';

export interface IconButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'secondary' | 'ghost' | 'primary';
  size?: 'sm' | 'md' | 'lg';
  /** Accessible label — required for icon-only controls. */
  title?: string;
  children?: React.ReactNode;
}

/** Square icon-only button. */
export function IconButton(props: IconButtonProps): JSX.Element;

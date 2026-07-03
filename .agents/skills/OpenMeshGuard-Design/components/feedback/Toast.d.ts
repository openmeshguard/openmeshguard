import * as React from 'react';

export interface ToastProps extends React.HTMLAttributes<HTMLDivElement> {
  status?: 'pass' | 'fail' | 'warn' | 'info';
  title?: React.ReactNode;
  message?: React.ReactNode;
  onDismiss?: () => void;
}

/** Toast notification with status accent. */
export function Toast(props: ToastProps): JSX.Element;

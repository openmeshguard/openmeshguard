import * as React from 'react';

export interface ProgressBarProps extends React.HTMLAttributes<HTMLDivElement> {
  /** 0–100. */
  value?: number;
  /** Override the auto threshold color (>=90 pass, >=60 warn, else fail). */
  status?: 'pass' | 'warn' | 'fail' | 'info' | 'none';
  showLabel?: boolean;
  /** Custom label; defaults to "NN%". */
  label?: React.ReactNode;
  height?: number;
}

/** Coverage / posture progress bar with threshold coloring. */
export function ProgressBar(props: ProgressBarProps): JSX.Element;

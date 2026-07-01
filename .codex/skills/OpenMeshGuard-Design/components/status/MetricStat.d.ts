import * as React from 'react';

export interface MetricStatProps extends React.HTMLAttributes<HTMLDivElement> {
  label: string;
  value: React.ReactNode;
  unit?: string;
  /** Secondary line, e.g. "+3 since last scan". */
  delta?: React.ReactNode;
  deltaTone?: 'positive' | 'negative' | 'neutral';
  /** Left accent bar color. */
  status?: 'pass' | 'warn' | 'fail' | 'info' | 'none';
}

/** KPI stat block for dashboard headers. */
export function MetricStat(props: MetricStatProps): JSX.Element;

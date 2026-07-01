import * as React from 'react';

export interface StatusBadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  /** Governance outcome. */
  status?: 'pass' | 'warn' | 'fail' | 'info' | 'none';
  /** Show the leading status dot (soft variant only). */
  dot?: boolean;
  /** Solid fill instead of soft tint. */
  solid?: boolean;
  children?: React.ReactNode;
}

/**
 * Status pill for control outcomes and lifecycle states.
 * @startingPoint section="Status" subtitle="Status pills, tags, coverage bars, metrics" viewport="700x220"
 */
export function StatusBadge(props: StatusBadgeProps): JSX.Element;

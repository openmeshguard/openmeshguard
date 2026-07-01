import * as React from 'react';

export interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  tone?: 'neutral' | 'brand' | 'outline';
  children?: React.ReactNode;
}

/** Small neutral label for counts and metadata (non-status). */
export function Badge(props: BadgeProps): JSX.Element;

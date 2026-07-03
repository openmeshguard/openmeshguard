import * as React from 'react';

export interface AvatarProps extends React.HTMLAttributes<HTMLSpanElement> {
  /** Full name; initials + stable color are derived from it. */
  name: string;
  size?: 'sm' | 'md' | 'lg';
}

/** Initials avatar for team owners. */
export function Avatar(props: AvatarProps): JSX.Element;

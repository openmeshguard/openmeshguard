import * as React from 'react';

export interface CardProps extends React.HTMLAttributes<HTMLElement> {
  title?: React.ReactNode;
  subtitle?: React.ReactNode;
  /** Header action buttons. */
  actions?: React.ReactNode;
  /** Apply default body padding (set false for tables/lists). */
  padded?: boolean;
  bodyStyle?: React.CSSProperties;
  children?: React.ReactNode;
}

/**
 * Surface container with optional header.
 * @startingPoint section="Data" subtitle="Cards, tables, tabs, avatars" viewport="700x320"
 */
export function Card(props: CardProps): JSX.Element;

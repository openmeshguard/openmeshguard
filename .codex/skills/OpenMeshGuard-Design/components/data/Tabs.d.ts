import * as React from 'react';

export interface TabItem { id: string; label: React.ReactNode; count?: number; }

export interface TabsProps extends React.HTMLAttributes<HTMLDivElement> {
  items: TabItem[];
  value: string;
  onChange?: (id: string) => void;
}

/** Underline tab bar with optional per-tab counts. */
export function Tabs(props: TabsProps): JSX.Element;

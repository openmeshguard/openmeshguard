import * as React from 'react';

export interface TagProps extends React.HTMLAttributes<HTMLSpanElement> {
  /** Optional key label, e.g. "owner" or "env". */
  label?: string;
  /** The value (rendered mono by default). */
  value: React.ReactNode;
  mono?: boolean;
  /** Show a remove button and handle its click. */
  onRemove?: () => void;
}

/** Metadata tag for owners, environments, namespaces, labels. */
export function Tag(props: TagProps): JSX.Element;

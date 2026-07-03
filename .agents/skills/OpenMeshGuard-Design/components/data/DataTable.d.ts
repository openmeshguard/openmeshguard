import * as React from 'react';

export interface DataTableColumn {
  key: string;
  header: React.ReactNode;
  /** Custom cell renderer: (value, row, index) => node. */
  render?: (value: any, row: any, index: number) => React.ReactNode;
  width?: string | number;
  align?: 'left' | 'center' | 'right';
  /** Render cell in mono (resource names, counts). */
  mono?: boolean;
}

export interface DataTableProps extends React.HTMLAttributes<HTMLDivElement> {
  columns: DataTableColumn[];
  rows: any[];
  onRowClick?: (row: any, index: number) => void;
  /** Field to use as React key. */
  rowKey?: string;
  empty?: React.ReactNode;
}

/** Dense governance data table with sticky header and row hover. */
export function DataTable(props: DataTableProps): JSX.Element;

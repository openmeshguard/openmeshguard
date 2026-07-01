import * as React from 'react';

export interface TooltipProps extends React.HTMLAttributes<HTMLSpanElement> {
  content: React.ReactNode;
  side?: 'top' | 'bottom' | 'left' | 'right';
  children: React.ReactNode;
}

/**
 * Hover tooltip wrapping a trigger.
 * @startingPoint section="Feedback" subtitle="Tooltip and toast" viewport="700x200"
 */
export function Tooltip(props: TooltipProps): JSX.Element;

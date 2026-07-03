import React from 'react';

/**
 * Governance status pill: pass / warn / fail / info / none.
 * Status is always carried by color + text (and optional dot), never color alone.
 */
export function StatusBadge({ status = 'none', children, dot = true, solid = false, style, ...rest }) {
  const base = {
    display: 'inline-flex', alignItems: 'center', gap: '6px',
    fontFamily: 'var(--font-sans)', fontSize: '12px', fontWeight: 600,
    lineHeight: 1, padding: '3px 9px', borderRadius: 'var(--radius-pill)',
    whiteSpace: 'nowrap',
  };
  const soft = {
    color: `var(--status-${status}-fg)`,
    background: `var(--status-${status}-bg)`,
    border: `1px solid var(--status-${status}-border)`,
  };
  const solidStyle = {
    color: status === 'warn' ? 'var(--slate-900)' : '#fff',
    background: `var(--status-${status}-solid)`,
    border: `1px solid var(--status-${status}-solid)`,
  };
  return (
    <span style={{ ...base, ...(solid ? solidStyle : soft), ...style }} {...rest}>
      {dot && !solid && (
        <span style={{ width: 7, height: 7, borderRadius: '50%', background: `var(--status-${status}-solid)`, flex: 'none' }} />
      )}
      {children}
    </span>
  );
}

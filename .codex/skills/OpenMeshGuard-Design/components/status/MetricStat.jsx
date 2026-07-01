import React from 'react';

/**
 * Big metric / KPI stat block for dashboard headers.
 * Value is mono; optional delta and status accent.
 */
export function MetricStat({ label, value, unit, delta, deltaTone = 'neutral', status, style, ...rest }) {
  const deltaColors = {
    positive: 'var(--status-pass-fg)',
    negative: 'var(--status-fail-fg)',
    neutral: 'var(--text-muted)',
  };
  return (
    <div
      style={{
        display: 'flex', flexDirection: 'column', gap: '6px',
        padding: 'var(--pad-card)', background: 'var(--surface-card)',
        border: '1px solid var(--border-subtle)', borderRadius: 'var(--radius-lg)',
        boxShadow: 'var(--shadow-xs)',
        borderLeft: status ? `3px solid var(--status-${status}-solid)` : '1px solid var(--border-subtle)',
        ...style,
      }}
      {...rest}
    >
      <span style={{ fontSize: '11px', fontWeight: 600, letterSpacing: '0.06em', textTransform: 'uppercase', color: 'var(--text-muted)' }}>{label}</span>
      <div style={{ display: 'flex', alignItems: 'baseline', gap: '4px' }}>
        <span style={{ fontFamily: 'var(--font-mono)', fontSize: '30px', fontWeight: 600, color: 'var(--text-strong)', lineHeight: 1 }}>{value}</span>
        {unit && <span style={{ fontFamily: 'var(--font-mono)', fontSize: '15px', color: 'var(--text-muted)' }}>{unit}</span>}
      </div>
      {delta != null && (
        <span style={{ fontSize: '12px', fontWeight: 500, color: deltaColors[deltaTone] }}>{delta}</span>
      )}
    </div>
  );
}

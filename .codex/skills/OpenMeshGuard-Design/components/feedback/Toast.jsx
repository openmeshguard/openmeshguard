import React from 'react';

const ICONS = {
  pass: <polyline points="20 6 9 17 4 12" />,
  fail: <g><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></g>,
  warn: <g><path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" /><line x1="12" y1="9" x2="12" y2="13" /><line x1="12" y1="17" x2="12.01" y2="17" /></g>,
  info: <g><circle cx="12" cy="12" r="10" /><line x1="12" y1="16" x2="12" y2="12" /><line x1="12" y1="8" x2="12.01" y2="8" /></g>,
};

/**
 * Toast notification. Fixed white surface with status accent and optional dismiss.
 */
export function Toast({ status = 'info', title, message, onDismiss, style, ...rest }) {
  return (
    <div
      role="status"
      style={{
        display: 'flex', alignItems: 'flex-start', gap: '10px',
        width: 360, maxWidth: '100%', padding: '12px 14px',
        background: 'var(--surface-card)', border: '1px solid var(--border-subtle)',
        borderLeft: `3px solid var(--status-${status}-solid)`,
        borderRadius: 'var(--radius-md)', boxShadow: 'var(--shadow-lg)',
        fontFamily: 'var(--font-sans)', ...style,
      }}
      {...rest}
    >
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke={`var(--status-${status}-solid)`}
        strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" style={{ flex: 'none', marginTop: 1 }}>
        {ICONS[status]}
      </svg>
      <div style={{ flex: 1, minWidth: 0 }}>
        {title && <div style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text-strong)' }}>{title}</div>}
        {message && <div style={{ fontSize: '13px', color: 'var(--text-body)', marginTop: title ? 2 : 0 }}>{message}</div>}
      </div>
      {onDismiss && (
        <button type="button" onClick={onDismiss} aria-label="Dismiss"
          style={{ border: 'none', background: 'none', cursor: 'pointer', color: 'var(--text-subtle)', padding: 0, fontSize: 16, lineHeight: 1, flex: 'none' }}>×</button>
      )}
    </div>
  );
}

import React from 'react';

/**
 * Surface container. Flat white with hairline border + xs shadow.
 * Optional header (title + actions) and padded body.
 */
export function Card({ title, subtitle, actions, children, padded = true, style, bodyStyle, ...rest }) {
  return (
    <section
      style={{
        background: 'var(--surface-card)', border: '1px solid var(--border-subtle)',
        borderRadius: 'var(--radius-lg)', boxShadow: 'var(--shadow-xs)', overflow: 'hidden', ...style,
      }}
      {...rest}
    >
      {(title || actions) && (
        <header style={{
          display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '12px',
          padding: '14px 20px', borderBottom: '1px solid var(--border-subtle)',
        }}>
          <div style={{ minWidth: 0 }}>
            {title && <h3 style={{ fontSize: '15px', fontWeight: 600, color: 'var(--text-strong)' }}>{title}</h3>}
            {subtitle && <p style={{ fontSize: '12px', color: 'var(--text-muted)', marginTop: 2 }}>{subtitle}</p>}
          </div>
          {actions && <div style={{ display: 'flex', gap: '8px', flex: 'none' }}>{actions}</div>}
        </header>
      )}
      <div style={{ padding: padded ? 'var(--pad-card)' : 0, ...bodyStyle }}>{children}</div>
    </section>
  );
}

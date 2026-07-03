import React from 'react';

/**
 * Underline tab bar. items: [{id,label,count?}]. Controlled via value/onChange.
 */
export function Tabs({ items = [], value, onChange, style, ...rest }) {
  const [hover, setHover] = React.useState(null);
  return (
    <div role="tablist" style={{ display: 'flex', gap: '4px', borderBottom: '1px solid var(--border-subtle)', ...style }} {...rest}>
      {items.map((it) => {
        const active = it.id === value;
        return (
          <button
            key={it.id} role="tab" aria-selected={active}
            onClick={() => onChange && onChange(it.id)}
            onMouseEnter={() => setHover(it.id)}
            onMouseLeave={() => setHover(null)}
            style={{
              display: 'inline-flex', alignItems: 'center', gap: '7px',
              background: 'none', border: 'none', cursor: 'pointer',
              padding: '9px 12px', marginBottom: '-1px',
              fontFamily: 'var(--font-sans)', fontSize: '14px',
              fontWeight: active ? 600 : 500,
              color: active ? 'var(--text-brand)' : hover === it.id ? 'var(--text-strong)' : 'var(--text-muted)',
              borderBottom: `2px solid ${active ? 'var(--brand-500)' : 'transparent'}`,
              transition: 'color var(--dur-fast)',
            }}
          >
            {it.label}
            {it.count != null && (
              <span style={{
                fontFamily: 'var(--font-mono)', fontSize: '11px', fontWeight: 600,
                padding: '1px 6px', borderRadius: 'var(--radius-pill)',
                background: active ? 'var(--brand-50)' : 'var(--surface-sunken)',
                color: active ? 'var(--brand-700)' : 'var(--text-muted)',
              }}>{it.count}</span>
            )}
          </button>
        );
      })}
    </div>
  );
}

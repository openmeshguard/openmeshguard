import React from 'react';

/**
 * Native select styled to match Input. Pass options as [{value,label}] or children.
 */
export function Select({ label, hint, options, id, size = 'md', style, containerStyle, disabled, children, ...rest }) {
  const [focus, setFocus] = React.useState(false);
  const selId = id || React.useId();
  const pad = size === 'sm' ? '5px 10px' : '8px 12px';
  const fs = size === 'sm' ? '13px' : '14px';

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '5px', ...containerStyle }}>
      {label && <label htmlFor={selId} style={{ fontSize: '13px', fontWeight: 500, color: 'var(--text-strong)' }}>{label}</label>}
      <div style={{ position: 'relative', display: 'flex' }}>
        <select
          id={selId}
          disabled={disabled}
          onFocus={() => setFocus(true)}
          onBlur={() => setFocus(false)}
          style={{
            appearance: 'none', width: '100%', fontFamily: 'var(--font-sans)', fontSize: fs,
            color: 'var(--text-strong)', padding: pad, paddingRight: '32px',
            background: disabled ? 'var(--surface-sunken)' : 'var(--surface-card)',
            border: `1px solid ${focus ? 'var(--border-brand)' : 'var(--border-default)'}`,
            borderRadius: 'var(--radius-md)', cursor: disabled ? 'not-allowed' : 'pointer',
            boxShadow: focus ? 'var(--shadow-focus)' : 'none', outline: 'none',
            transition: 'border-color var(--dur-fast), box-shadow var(--dur-fast)', ...style,
          }}
          {...rest}
        >
          {options ? options.map((o) => <option key={o.value} value={o.value}>{o.label}</option>) : children}
        </select>
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"
          style={{ position: 'absolute', right: 10, top: '50%', transform: 'translateY(-50%)', color: 'var(--text-subtle)', pointerEvents: 'none' }}>
          <polyline points="6 9 12 15 18 9" />
        </svg>
      </div>
      {hint && <span style={{ fontSize: '12px', color: 'var(--text-muted)' }}>{hint}</span>}
    </div>
  );
}

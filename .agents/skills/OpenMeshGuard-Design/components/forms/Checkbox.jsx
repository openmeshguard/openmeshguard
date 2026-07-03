import React from 'react';

/**
 * Checkbox with label. Controlled via `checked` / `onChange`.
 */
export function Checkbox({ checked = false, onChange, label, disabled = false, id, style, ...rest }) {
  const cbId = id || React.useId();
  return (
    <label
      htmlFor={cbId}
      style={{
        display: 'inline-flex', alignItems: 'center', gap: '8px',
        cursor: disabled ? 'not-allowed' : 'pointer', opacity: disabled ? 0.5 : 1,
        fontFamily: 'var(--font-sans)', fontSize: '14px', color: 'var(--text-body)', ...style,
      }}
    >
      <span style={{
        width: 16, height: 16, flex: 'none', borderRadius: 'var(--radius-xs)',
        border: `1.5px solid ${checked ? 'var(--brand-500)' : 'var(--border-strong)'}`,
        background: checked ? 'var(--brand-500)' : 'var(--surface-card)',
        display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
        transition: 'background var(--dur-fast), border-color var(--dur-fast)',
      }}>
        {checked && (
          <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="#fff" strokeWidth="3.5" strokeLinecap="round" strokeLinejoin="round">
            <polyline points="20 6 9 17 4 12" />
          </svg>
        )}
      </span>
      <input id={cbId} type="checkbox" checked={checked} onChange={onChange} disabled={disabled}
        style={{ position: 'absolute', opacity: 0, width: 0, height: 0 }} {...rest} />
      {label && <span>{label}</span>}
    </label>
  );
}

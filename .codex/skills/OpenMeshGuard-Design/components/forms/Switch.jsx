import React from 'react';

/**
 * Toggle switch for on/off settings (e.g. enable a control, mute a finding).
 */
export function Switch({ checked = false, onChange, label, disabled = false, id, style, ...rest }) {
  const swId = id || React.useId();
  return (
    <label
      htmlFor={swId}
      style={{
        display: 'inline-flex', alignItems: 'center', gap: '10px',
        cursor: disabled ? 'not-allowed' : 'pointer', opacity: disabled ? 0.5 : 1,
        fontFamily: 'var(--font-sans)', fontSize: '14px', color: 'var(--text-body)', ...style,
      }}
    >
      <span
        onClick={() => !disabled && onChange && onChange(!checked)}
        style={{
          width: 34, height: 20, flex: 'none', borderRadius: 'var(--radius-pill)',
          background: checked ? 'var(--brand-500)' : 'var(--slate-300)',
          position: 'relative', transition: 'background var(--dur-normal) var(--ease-standard)',
        }}
      >
        <span style={{
          position: 'absolute', top: 2, left: checked ? 16 : 2, width: 16, height: 16,
          borderRadius: '50%', background: '#fff', boxShadow: 'var(--shadow-sm)',
          transition: 'left var(--dur-normal) var(--ease-out)',
        }} />
      </span>
      <input id={swId} type="checkbox" checked={checked} disabled={disabled}
        onChange={(e) => onChange && onChange(e.target.checked)}
        style={{ position: 'absolute', opacity: 0, width: 0, height: 0 }} {...rest} />
      {label && <span>{label}</span>}
    </label>
  );
}

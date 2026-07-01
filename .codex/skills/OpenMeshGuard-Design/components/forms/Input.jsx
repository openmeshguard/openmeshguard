import React from 'react';

/**
 * Text input with optional label, hint, error, and leading icon/adornment.
 */
export function Input({
  label, hint, error, leadingIcon, id, size = 'md', style, containerStyle, disabled, ...rest
}) {
  const [focus, setFocus] = React.useState(false);
  const inputId = id || React.useId();
  const pad = size === 'sm' ? '5px 10px' : '8px 12px';
  const fs = size === 'sm' ? '13px' : '14px';
  const borderColor = error ? 'var(--status-fail-solid)' : focus ? 'var(--border-brand)' : 'var(--border-default)';

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '5px', ...containerStyle }}>
      {label && (
        <label htmlFor={inputId} style={{ fontSize: '13px', fontWeight: 500, color: 'var(--text-strong)' }}>{label}</label>
      )}
      <div style={{
        display: 'flex', alignItems: 'center', gap: '8px',
        background: disabled ? 'var(--surface-sunken)' : 'var(--surface-card)',
        border: `1px solid ${borderColor}`, borderRadius: 'var(--radius-md)',
        boxShadow: focus ? 'var(--shadow-focus)' : 'none',
        padding: `0 ${pad.split(' ')[1]}`,
        transition: 'border-color var(--dur-fast), box-shadow var(--dur-fast)',
      }}>
        {leadingIcon && <span style={{ display: 'inline-flex', color: 'var(--text-subtle)', flex: 'none' }}>{leadingIcon}</span>}
        <input
          id={inputId}
          disabled={disabled}
          onFocus={() => setFocus(true)}
          onBlur={() => setFocus(false)}
          style={{
            flex: 1, border: 'none', outline: 'none', background: 'transparent',
            fontFamily: 'var(--font-sans)', fontSize: fs, color: 'var(--text-strong)',
            padding: `${pad.split(' ')[0]} 0`, minWidth: 0, ...style,
          }}
          {...rest}
        />
      </div>
      {(hint || error) && (
        <span style={{ fontSize: '12px', color: error ? 'var(--status-fail-fg)' : 'var(--text-muted)' }}>{error || hint}</span>
      )}
    </div>
  );
}

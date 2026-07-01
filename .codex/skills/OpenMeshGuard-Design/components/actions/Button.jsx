import React from 'react';

const SIZES = {
  sm: { fontSize: '13px', padding: '5px 10px', height: '30px', gap: '6px' },
  md: { fontSize: '14px', padding: '7px 14px', height: '36px', gap: '7px' },
  lg: { fontSize: '15px', padding: '9px 18px', height: '42px', gap: '8px' },
};

const VARIANTS = {
  primary: {
    background: 'var(--action-bg)', color: 'var(--action-fg)',
    border: '1px solid var(--action-bg)',
    hoverBg: 'var(--action-bg-hover)', activeBg: 'var(--action-bg-active)',
  },
  secondary: {
    background: 'var(--surface-card)', color: 'var(--text-strong)',
    border: '1px solid var(--border-default)',
    hoverBg: 'var(--surface-hover)', activeBg: 'var(--surface-active)',
  },
  ghost: {
    background: 'transparent', color: 'var(--text-body)',
    border: '1px solid transparent',
    hoverBg: 'var(--surface-hover)', activeBg: 'var(--surface-active)',
  },
  danger: {
    background: 'var(--status-fail-solid)', color: '#fff',
    border: '1px solid var(--status-fail-solid)',
    hoverBg: 'var(--red-700)', activeBg: 'var(--red-800)',
  },
};

/**
 * OpenMeshGuard primary button. Sentence-case labels, no icons required.
 */
export function Button({
  variant = 'primary',
  size = 'md',
  leftIcon,
  rightIcon,
  fullWidth = false,
  disabled = false,
  type = 'button',
  children,
  style,
  ...rest
}) {
  const v = VARIANTS[variant] || VARIANTS.primary;
  const s = SIZES[size] || SIZES.md;
  const [hover, setHover] = React.useState(false);
  const [active, setActive] = React.useState(false);

  const bg = disabled ? v.background : (active ? v.activeBg : hover ? v.hoverBg : v.background);

  return (
    <button
      type={type}
      disabled={disabled}
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => { setHover(false); setActive(false); }}
      onMouseDown={() => setActive(true)}
      onMouseUp={() => setActive(false)}
      style={{
        display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
        gap: s.gap, fontFamily: 'var(--font-sans)', fontWeight: 600,
        fontSize: s.fontSize, lineHeight: 1, padding: s.padding, height: s.height,
        width: fullWidth ? '100%' : 'auto',
        color: v.color, background: bg, border: v.border,
        borderRadius: 'var(--radius-md)', cursor: disabled ? 'not-allowed' : 'pointer',
        opacity: disabled ? 0.5 : 1,
        transition: 'background var(--dur-fast) var(--ease-standard)',
        whiteSpace: 'nowrap', ...style,
      }}
      {...rest}
    >
      {leftIcon && <span style={{ display: 'inline-flex', flex: 'none' }}>{leftIcon}</span>}
      {children}
      {rightIcon && <span style={{ display: 'inline-flex', flex: 'none' }}>{rightIcon}</span>}
    </button>
  );
}

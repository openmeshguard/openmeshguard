import React from 'react';

const SIZES = { sm: 30, md: 36, lg: 42 };

/**
 * Square icon-only button. Always pass an accessible `title` / aria-label.
 */
export function IconButton({
  variant = 'secondary',
  size = 'md',
  disabled = false,
  title,
  children,
  style,
  ...rest
}) {
  const dim = SIZES[size] || SIZES.md;
  const [hover, setHover] = React.useState(false);

  const variants = {
    secondary: { bg: 'var(--surface-card)', border: '1px solid var(--border-default)', color: 'var(--text-body)', hoverBg: 'var(--surface-hover)' },
    ghost: { bg: 'transparent', border: '1px solid transparent', color: 'var(--text-muted)', hoverBg: 'var(--surface-hover)' },
    primary: { bg: 'var(--action-bg)', border: '1px solid var(--action-bg)', color: '#fff', hoverBg: 'var(--action-bg-hover)' },
  };
  const v = variants[variant] || variants.secondary;

  return (
    <button
      type="button"
      title={title}
      aria-label={title}
      disabled={disabled}
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
      style={{
        display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
        width: dim, height: dim, flex: 'none',
        background: disabled ? v.bg : (hover ? v.hoverBg : v.bg),
        border: v.border, color: v.color, borderRadius: 'var(--radius-md)',
        cursor: disabled ? 'not-allowed' : 'pointer', opacity: disabled ? 0.5 : 1,
        transition: 'background var(--dur-fast) var(--ease-standard)', ...style,
      }}
      {...rest}
    >
      {children}
    </button>
  );
}

import React from 'react';

/**
 * Lightweight hover tooltip. Wraps its trigger children.
 */
export function Tooltip({ content, side = 'top', children, style, ...rest }) {
  const [show, setShow] = React.useState(false);
  const pos = {
    top: { bottom: '100%', left: '50%', transform: 'translateX(-50%)', marginBottom: 6 },
    bottom: { top: '100%', left: '50%', transform: 'translateX(-50%)', marginTop: 6 },
    left: { right: '100%', top: '50%', transform: 'translateY(-50%)', marginRight: 6 },
    right: { left: '100%', top: '50%', transform: 'translateY(-50%)', marginLeft: 6 },
  }[side];
  return (
    <span
      style={{ position: 'relative', display: 'inline-flex' }}
      onMouseEnter={() => setShow(true)}
      onMouseLeave={() => setShow(false)}
      {...rest}
    >
      {children}
      {show && content && (
        <span
          role="tooltip"
          style={{
            position: 'absolute', zIndex: 50, ...pos, whiteSpace: 'nowrap',
            background: 'var(--slate-900)', color: 'var(--slate-0)',
            fontFamily: 'var(--font-sans)', fontSize: '12px', fontWeight: 500,
            padding: '5px 9px', borderRadius: 'var(--radius-sm)',
            boxShadow: 'var(--shadow-md)', pointerEvents: 'none', ...style,
          }}
        >{content}</span>
      )}
    </span>
  );
}

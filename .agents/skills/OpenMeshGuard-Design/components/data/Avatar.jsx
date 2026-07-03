import React from 'react';

const SIZES = { sm: 24, md: 32, lg: 40 };
const PALETTE = ['--brand-500', '--emerald-600', '--info-600', '--amber-600', '--slate-600'];

function initials(name = '') {
  return name.trim().split(/\s+/).slice(0, 2).map((w) => w[0]).join('').toUpperCase() || '?';
}

/**
 * Initials avatar for team owners. Color derived from name for stable identity.
 */
export function Avatar({ name = '', size = 'md', style, ...rest }) {
  const dim = SIZES[size] || SIZES.md;
  const idx = [...name].reduce((a, c) => a + c.charCodeAt(0), 0) % PALETTE.length;
  return (
    <span
      title={name}
      style={{
        display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
        width: dim, height: dim, flex: 'none', borderRadius: '50%',
        background: `var(${PALETTE[idx]})`, color: '#fff',
        fontFamily: 'var(--font-sans)', fontWeight: 600, fontSize: dim * 0.4,
        letterSpacing: '0.02em', ...style,
      }}
      {...rest}
    >
      {initials(name)}
    </span>
  );
}

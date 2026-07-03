/* @ds-bundle: {"format":3,"namespace":"OpenMeshGuardDesignSystem_65348c","components":[{"name":"Button","sourcePath":"components/actions/Button.jsx"},{"name":"IconButton","sourcePath":"components/actions/IconButton.jsx"},{"name":"Avatar","sourcePath":"components/data/Avatar.jsx"},{"name":"Card","sourcePath":"components/data/Card.jsx"},{"name":"DataTable","sourcePath":"components/data/DataTable.jsx"},{"name":"Tabs","sourcePath":"components/data/Tabs.jsx"},{"name":"Toast","sourcePath":"components/feedback/Toast.jsx"},{"name":"Tooltip","sourcePath":"components/feedback/Tooltip.jsx"},{"name":"Checkbox","sourcePath":"components/forms/Checkbox.jsx"},{"name":"Input","sourcePath":"components/forms/Input.jsx"},{"name":"Select","sourcePath":"components/forms/Select.jsx"},{"name":"Switch","sourcePath":"components/forms/Switch.jsx"},{"name":"Badge","sourcePath":"components/status/Badge.jsx"},{"name":"MetricStat","sourcePath":"components/status/MetricStat.jsx"},{"name":"ProgressBar","sourcePath":"components/status/ProgressBar.jsx"},{"name":"StatusBadge","sourcePath":"components/status/StatusBadge.jsx"},{"name":"Tag","sourcePath":"components/status/Tag.jsx"}],"sourceHashes":{"components/actions/Button.jsx":"c13540b69993","components/actions/IconButton.jsx":"857dc3856dc9","components/data/Avatar.jsx":"9613bc236b07","components/data/Card.jsx":"24b2d02a7295","components/data/DataTable.jsx":"fcc1944755da","components/data/Tabs.jsx":"500ded393c1f","components/feedback/Toast.jsx":"30651a6a5d3f","components/feedback/Tooltip.jsx":"55327744cfdd","components/forms/Checkbox.jsx":"60260dd057bd","components/forms/Input.jsx":"ee80e42ce48a","components/forms/Select.jsx":"27dfee2eb04a","components/forms/Switch.jsx":"d4b172a28937","components/status/Badge.jsx":"6d35e84d1c4c","components/status/MetricStat.jsx":"f972975b142a","components/status/ProgressBar.jsx":"af6e055aa3d9","components/status/StatusBadge.jsx":"509e730d0d4b","components/status/Tag.jsx":"736ba8886c57","ui_kits/posture-scanner/AppShell.jsx":"785cc123111b","ui_kits/posture-scanner/Evidence.jsx":"fe61e3d3e74f","ui_kits/posture-scanner/Findings.jsx":"b7019b355f03","ui_kits/posture-scanner/Icons.jsx":"b6d60e683465","ui_kits/posture-scanner/Overview.jsx":"3a2162e658d8","ui_kits/posture-scanner/ResourceDetail.jsx":"1f7830b07553","ui_kits/posture-scanner/Workloads.jsx":"86dbfb873d50","ui_kits/posture-scanner/app.jsx":"286ce84c3579","ui_kits/posture-scanner/data.jsx":"ede00a3eb110"},"inlinedExternals":[],"unexposedExports":[]} */

(() => {

const __ds_ns = (window.OpenMeshGuardDesignSystem_65348c = window.OpenMeshGuardDesignSystem_65348c || {});

const __ds_scope = {};

(__ds_ns.__errors = __ds_ns.__errors || []);

// components/actions/Button.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
const SIZES = {
  sm: {
    fontSize: '13px',
    padding: '5px 10px',
    height: '30px',
    gap: '6px'
  },
  md: {
    fontSize: '14px',
    padding: '7px 14px',
    height: '36px',
    gap: '7px'
  },
  lg: {
    fontSize: '15px',
    padding: '9px 18px',
    height: '42px',
    gap: '8px'
  }
};
const VARIANTS = {
  primary: {
    background: 'var(--action-bg)',
    color: 'var(--action-fg)',
    border: '1px solid var(--action-bg)',
    hoverBg: 'var(--action-bg-hover)',
    activeBg: 'var(--action-bg-active)'
  },
  secondary: {
    background: 'var(--surface-card)',
    color: 'var(--text-strong)',
    border: '1px solid var(--border-default)',
    hoverBg: 'var(--surface-hover)',
    activeBg: 'var(--surface-active)'
  },
  ghost: {
    background: 'transparent',
    color: 'var(--text-body)',
    border: '1px solid transparent',
    hoverBg: 'var(--surface-hover)',
    activeBg: 'var(--surface-active)'
  },
  danger: {
    background: 'var(--status-fail-solid)',
    color: '#fff',
    border: '1px solid var(--status-fail-solid)',
    hoverBg: 'var(--red-700)',
    activeBg: 'var(--red-800)'
  }
};

/**
 * OpenMeshGuard primary button. Sentence-case labels, no icons required.
 */
function Button({
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
  const bg = disabled ? v.background : active ? v.activeBg : hover ? v.hoverBg : v.background;
  return /*#__PURE__*/React.createElement("button", _extends({
    type: type,
    disabled: disabled,
    onMouseEnter: () => setHover(true),
    onMouseLeave: () => {
      setHover(false);
      setActive(false);
    },
    onMouseDown: () => setActive(true),
    onMouseUp: () => setActive(false),
    style: {
      display: 'inline-flex',
      alignItems: 'center',
      justifyContent: 'center',
      gap: s.gap,
      fontFamily: 'var(--font-sans)',
      fontWeight: 600,
      fontSize: s.fontSize,
      lineHeight: 1,
      padding: s.padding,
      height: s.height,
      width: fullWidth ? '100%' : 'auto',
      color: v.color,
      background: bg,
      border: v.border,
      borderRadius: 'var(--radius-md)',
      cursor: disabled ? 'not-allowed' : 'pointer',
      opacity: disabled ? 0.5 : 1,
      transition: 'background var(--dur-fast) var(--ease-standard)',
      whiteSpace: 'nowrap',
      ...style
    }
  }, rest), leftIcon && /*#__PURE__*/React.createElement("span", {
    style: {
      display: 'inline-flex',
      flex: 'none'
    }
  }, leftIcon), children, rightIcon && /*#__PURE__*/React.createElement("span", {
    style: {
      display: 'inline-flex',
      flex: 'none'
    }
  }, rightIcon));
}
Object.assign(__ds_scope, { Button });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/actions/Button.jsx", error: String((e && e.message) || e) }); }

// components/actions/IconButton.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
const SIZES = {
  sm: 30,
  md: 36,
  lg: 42
};

/**
 * Square icon-only button. Always pass an accessible `title` / aria-label.
 */
function IconButton({
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
    secondary: {
      bg: 'var(--surface-card)',
      border: '1px solid var(--border-default)',
      color: 'var(--text-body)',
      hoverBg: 'var(--surface-hover)'
    },
    ghost: {
      bg: 'transparent',
      border: '1px solid transparent',
      color: 'var(--text-muted)',
      hoverBg: 'var(--surface-hover)'
    },
    primary: {
      bg: 'var(--action-bg)',
      border: '1px solid var(--action-bg)',
      color: '#fff',
      hoverBg: 'var(--action-bg-hover)'
    }
  };
  const v = variants[variant] || variants.secondary;
  return /*#__PURE__*/React.createElement("button", _extends({
    type: "button",
    title: title,
    "aria-label": title,
    disabled: disabled,
    onMouseEnter: () => setHover(true),
    onMouseLeave: () => setHover(false),
    style: {
      display: 'inline-flex',
      alignItems: 'center',
      justifyContent: 'center',
      width: dim,
      height: dim,
      flex: 'none',
      background: disabled ? v.bg : hover ? v.hoverBg : v.bg,
      border: v.border,
      color: v.color,
      borderRadius: 'var(--radius-md)',
      cursor: disabled ? 'not-allowed' : 'pointer',
      opacity: disabled ? 0.5 : 1,
      transition: 'background var(--dur-fast) var(--ease-standard)',
      ...style
    }
  }, rest), children);
}
Object.assign(__ds_scope, { IconButton });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/actions/IconButton.jsx", error: String((e && e.message) || e) }); }

// components/data/Avatar.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
const SIZES = {
  sm: 24,
  md: 32,
  lg: 40
};
const PALETTE = ['--brand-500', '--emerald-600', '--info-600', '--amber-600', '--slate-600'];
function initials(name = '') {
  return name.trim().split(/\s+/).slice(0, 2).map(w => w[0]).join('').toUpperCase() || '?';
}

/**
 * Initials avatar for team owners. Color derived from name for stable identity.
 */
function Avatar({
  name = '',
  size = 'md',
  style,
  ...rest
}) {
  const dim = SIZES[size] || SIZES.md;
  const idx = [...name].reduce((a, c) => a + c.charCodeAt(0), 0) % PALETTE.length;
  return /*#__PURE__*/React.createElement("span", _extends({
    title: name,
    style: {
      display: 'inline-flex',
      alignItems: 'center',
      justifyContent: 'center',
      width: dim,
      height: dim,
      flex: 'none',
      borderRadius: '50%',
      background: `var(${PALETTE[idx]})`,
      color: '#fff',
      fontFamily: 'var(--font-sans)',
      fontWeight: 600,
      fontSize: dim * 0.4,
      letterSpacing: '0.02em',
      ...style
    }
  }, rest), initials(name));
}
Object.assign(__ds_scope, { Avatar });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/data/Avatar.jsx", error: String((e && e.message) || e) }); }

// components/data/Card.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/**
 * Surface container. Flat white with hairline border + xs shadow.
 * Optional header (title + actions) and padded body.
 */
function Card({
  title,
  subtitle,
  actions,
  children,
  padded = true,
  style,
  bodyStyle,
  ...rest
}) {
  return /*#__PURE__*/React.createElement("section", _extends({
    style: {
      background: 'var(--surface-card)',
      border: '1px solid var(--border-subtle)',
      borderRadius: 'var(--radius-lg)',
      boxShadow: 'var(--shadow-xs)',
      overflow: 'hidden',
      ...style
    }
  }, rest), (title || actions) && /*#__PURE__*/React.createElement("header", {
    style: {
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'space-between',
      gap: '12px',
      padding: '14px 20px',
      borderBottom: '1px solid var(--border-subtle)'
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      minWidth: 0
    }
  }, title && /*#__PURE__*/React.createElement("h3", {
    style: {
      fontSize: '15px',
      fontWeight: 600,
      color: 'var(--text-strong)'
    }
  }, title), subtitle && /*#__PURE__*/React.createElement("p", {
    style: {
      fontSize: '12px',
      color: 'var(--text-muted)',
      marginTop: 2
    }
  }, subtitle)), actions && /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      gap: '8px',
      flex: 'none'
    }
  }, actions)), /*#__PURE__*/React.createElement("div", {
    style: {
      padding: padded ? 'var(--pad-card)' : 0,
      ...bodyStyle
    }
  }, children));
}
Object.assign(__ds_scope, { Card });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/data/Card.jsx", error: String((e && e.message) || e) }); }

// components/data/DataTable.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/**
 * Dense governance data table. Columns: [{key,header,render?,width?,align?,mono?}].
 * Rows are plain objects. Optional row click, hover highlight, and sticky header.
 */
function DataTable({
  columns = [],
  rows = [],
  onRowClick,
  rowKey,
  empty = 'No data.',
  style,
  ...rest
}) {
  const [hover, setHover] = React.useState(-1);
  return /*#__PURE__*/React.createElement("div", _extends({
    style: {
      width: '100%',
      overflowX: 'auto',
      ...style
    }
  }, rest), /*#__PURE__*/React.createElement("table", {
    style: {
      width: '100%',
      borderCollapse: 'collapse',
      fontFamily: 'var(--font-sans)'
    }
  }, /*#__PURE__*/React.createElement("thead", null, /*#__PURE__*/React.createElement("tr", null, columns.map(c => /*#__PURE__*/React.createElement("th", {
    key: c.key,
    style: {
      textAlign: c.align || 'left',
      padding: 'var(--pad-cell-y) var(--pad-cell-x)',
      fontSize: '11px',
      fontWeight: 600,
      letterSpacing: '0.05em',
      textTransform: 'uppercase',
      color: 'var(--text-muted)',
      background: 'var(--surface-sunken)',
      borderBottom: '1px solid var(--border-subtle)',
      whiteSpace: 'nowrap',
      width: c.width,
      position: 'sticky',
      top: 0
    }
  }, c.header)))), /*#__PURE__*/React.createElement("tbody", null, rows.length === 0 && /*#__PURE__*/React.createElement("tr", null, /*#__PURE__*/React.createElement("td", {
    colSpan: columns.length,
    style: {
      padding: '28px',
      textAlign: 'center',
      color: 'var(--text-muted)',
      fontSize: '13px'
    }
  }, empty)), rows.map((row, i) => /*#__PURE__*/React.createElement("tr", {
    key: rowKey ? row[rowKey] : i,
    onMouseEnter: () => setHover(i),
    onMouseLeave: () => setHover(-1),
    onClick: onRowClick ? () => onRowClick(row, i) : undefined,
    style: {
      background: hover === i ? 'var(--surface-hover)' : 'transparent',
      cursor: onRowClick ? 'pointer' : 'default',
      transition: 'background var(--dur-fast)'
    }
  }, columns.map(c => /*#__PURE__*/React.createElement("td", {
    key: c.key,
    style: {
      padding: 'var(--pad-cell-y) var(--pad-cell-x)',
      textAlign: c.align || 'left',
      fontSize: c.mono ? '13px' : '13px',
      fontFamily: c.mono ? 'var(--font-mono)' : 'var(--font-sans)',
      color: c.mono ? 'var(--text-strong)' : 'var(--text-body)',
      borderBottom: '1px solid var(--border-subtle)',
      whiteSpace: 'nowrap'
    }
  }, c.render ? c.render(row[c.key], row, i) : row[c.key])))))));
}
Object.assign(__ds_scope, { DataTable });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/data/DataTable.jsx", error: String((e && e.message) || e) }); }

// components/data/Tabs.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/**
 * Underline tab bar. items: [{id,label,count?}]. Controlled via value/onChange.
 */
function Tabs({
  items = [],
  value,
  onChange,
  style,
  ...rest
}) {
  const [hover, setHover] = React.useState(null);
  return /*#__PURE__*/React.createElement("div", _extends({
    role: "tablist",
    style: {
      display: 'flex',
      gap: '4px',
      borderBottom: '1px solid var(--border-subtle)',
      ...style
    }
  }, rest), items.map(it => {
    const active = it.id === value;
    return /*#__PURE__*/React.createElement("button", {
      key: it.id,
      role: "tab",
      "aria-selected": active,
      onClick: () => onChange && onChange(it.id),
      onMouseEnter: () => setHover(it.id),
      onMouseLeave: () => setHover(null),
      style: {
        display: 'inline-flex',
        alignItems: 'center',
        gap: '7px',
        background: 'none',
        border: 'none',
        cursor: 'pointer',
        padding: '9px 12px',
        marginBottom: '-1px',
        fontFamily: 'var(--font-sans)',
        fontSize: '14px',
        fontWeight: active ? 600 : 500,
        color: active ? 'var(--text-brand)' : hover === it.id ? 'var(--text-strong)' : 'var(--text-muted)',
        borderBottom: `2px solid ${active ? 'var(--brand-500)' : 'transparent'}`,
        transition: 'color var(--dur-fast)'
      }
    }, it.label, it.count != null && /*#__PURE__*/React.createElement("span", {
      style: {
        fontFamily: 'var(--font-mono)',
        fontSize: '11px',
        fontWeight: 600,
        padding: '1px 6px',
        borderRadius: 'var(--radius-pill)',
        background: active ? 'var(--brand-50)' : 'var(--surface-sunken)',
        color: active ? 'var(--brand-700)' : 'var(--text-muted)'
      }
    }, it.count));
  }));
}
Object.assign(__ds_scope, { Tabs });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/data/Tabs.jsx", error: String((e && e.message) || e) }); }

// components/feedback/Toast.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
const ICONS = {
  pass: /*#__PURE__*/React.createElement("polyline", {
    points: "20 6 9 17 4 12"
  }),
  fail: /*#__PURE__*/React.createElement("g", null, /*#__PURE__*/React.createElement("line", {
    x1: "18",
    y1: "6",
    x2: "6",
    y2: "18"
  }), /*#__PURE__*/React.createElement("line", {
    x1: "6",
    y1: "6",
    x2: "18",
    y2: "18"
  })),
  warn: /*#__PURE__*/React.createElement("g", null, /*#__PURE__*/React.createElement("path", {
    d: "M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"
  }), /*#__PURE__*/React.createElement("line", {
    x1: "12",
    y1: "9",
    x2: "12",
    y2: "13"
  }), /*#__PURE__*/React.createElement("line", {
    x1: "12",
    y1: "17",
    x2: "12.01",
    y2: "17"
  })),
  info: /*#__PURE__*/React.createElement("g", null, /*#__PURE__*/React.createElement("circle", {
    cx: "12",
    cy: "12",
    r: "10"
  }), /*#__PURE__*/React.createElement("line", {
    x1: "12",
    y1: "16",
    x2: "12",
    y2: "12"
  }), /*#__PURE__*/React.createElement("line", {
    x1: "12",
    y1: "8",
    x2: "12.01",
    y2: "8"
  }))
};

/**
 * Toast notification. Fixed white surface with status accent and optional dismiss.
 */
function Toast({
  status = 'info',
  title,
  message,
  onDismiss,
  style,
  ...rest
}) {
  return /*#__PURE__*/React.createElement("div", _extends({
    role: "status",
    style: {
      display: 'flex',
      alignItems: 'flex-start',
      gap: '10px',
      width: 360,
      maxWidth: '100%',
      padding: '12px 14px',
      background: 'var(--surface-card)',
      border: '1px solid var(--border-subtle)',
      borderLeft: `3px solid var(--status-${status}-solid)`,
      borderRadius: 'var(--radius-md)',
      boxShadow: 'var(--shadow-lg)',
      fontFamily: 'var(--font-sans)',
      ...style
    }
  }, rest), /*#__PURE__*/React.createElement("svg", {
    width: "18",
    height: "18",
    viewBox: "0 0 24 24",
    fill: "none",
    stroke: `var(--status-${status}-solid)`,
    strokeWidth: "2.2",
    strokeLinecap: "round",
    strokeLinejoin: "round",
    style: {
      flex: 'none',
      marginTop: 1
    }
  }, ICONS[status]), /*#__PURE__*/React.createElement("div", {
    style: {
      flex: 1,
      minWidth: 0
    }
  }, title && /*#__PURE__*/React.createElement("div", {
    style: {
      fontSize: '13px',
      fontWeight: 600,
      color: 'var(--text-strong)'
    }
  }, title), message && /*#__PURE__*/React.createElement("div", {
    style: {
      fontSize: '13px',
      color: 'var(--text-body)',
      marginTop: title ? 2 : 0
    }
  }, message)), onDismiss && /*#__PURE__*/React.createElement("button", {
    type: "button",
    onClick: onDismiss,
    "aria-label": "Dismiss",
    style: {
      border: 'none',
      background: 'none',
      cursor: 'pointer',
      color: 'var(--text-subtle)',
      padding: 0,
      fontSize: 16,
      lineHeight: 1,
      flex: 'none'
    }
  }, "\xD7"));
}
Object.assign(__ds_scope, { Toast });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/feedback/Toast.jsx", error: String((e && e.message) || e) }); }

// components/feedback/Tooltip.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/**
 * Lightweight hover tooltip. Wraps its trigger children.
 */
function Tooltip({
  content,
  side = 'top',
  children,
  style,
  ...rest
}) {
  const [show, setShow] = React.useState(false);
  const pos = {
    top: {
      bottom: '100%',
      left: '50%',
      transform: 'translateX(-50%)',
      marginBottom: 6
    },
    bottom: {
      top: '100%',
      left: '50%',
      transform: 'translateX(-50%)',
      marginTop: 6
    },
    left: {
      right: '100%',
      top: '50%',
      transform: 'translateY(-50%)',
      marginRight: 6
    },
    right: {
      left: '100%',
      top: '50%',
      transform: 'translateY(-50%)',
      marginLeft: 6
    }
  }[side];
  return /*#__PURE__*/React.createElement("span", _extends({
    style: {
      position: 'relative',
      display: 'inline-flex'
    },
    onMouseEnter: () => setShow(true),
    onMouseLeave: () => setShow(false)
  }, rest), children, show && content && /*#__PURE__*/React.createElement("span", {
    role: "tooltip",
    style: {
      position: 'absolute',
      zIndex: 50,
      ...pos,
      whiteSpace: 'nowrap',
      background: 'var(--slate-900)',
      color: 'var(--slate-0)',
      fontFamily: 'var(--font-sans)',
      fontSize: '12px',
      fontWeight: 500,
      padding: '5px 9px',
      borderRadius: 'var(--radius-sm)',
      boxShadow: 'var(--shadow-md)',
      pointerEvents: 'none',
      ...style
    }
  }, content));
}
Object.assign(__ds_scope, { Tooltip });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/feedback/Tooltip.jsx", error: String((e && e.message) || e) }); }

// components/forms/Checkbox.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/**
 * Checkbox with label. Controlled via `checked` / `onChange`.
 */
function Checkbox({
  checked = false,
  onChange,
  label,
  disabled = false,
  id,
  style,
  ...rest
}) {
  const cbId = id || React.useId();
  return /*#__PURE__*/React.createElement("label", {
    htmlFor: cbId,
    style: {
      display: 'inline-flex',
      alignItems: 'center',
      gap: '8px',
      cursor: disabled ? 'not-allowed' : 'pointer',
      opacity: disabled ? 0.5 : 1,
      fontFamily: 'var(--font-sans)',
      fontSize: '14px',
      color: 'var(--text-body)',
      ...style
    }
  }, /*#__PURE__*/React.createElement("span", {
    style: {
      width: 16,
      height: 16,
      flex: 'none',
      borderRadius: 'var(--radius-xs)',
      border: `1.5px solid ${checked ? 'var(--brand-500)' : 'var(--border-strong)'}`,
      background: checked ? 'var(--brand-500)' : 'var(--surface-card)',
      display: 'inline-flex',
      alignItems: 'center',
      justifyContent: 'center',
      transition: 'background var(--dur-fast), border-color var(--dur-fast)'
    }
  }, checked && /*#__PURE__*/React.createElement("svg", {
    width: "11",
    height: "11",
    viewBox: "0 0 24 24",
    fill: "none",
    stroke: "#fff",
    strokeWidth: "3.5",
    strokeLinecap: "round",
    strokeLinejoin: "round"
  }, /*#__PURE__*/React.createElement("polyline", {
    points: "20 6 9 17 4 12"
  }))), /*#__PURE__*/React.createElement("input", _extends({
    id: cbId,
    type: "checkbox",
    checked: checked,
    onChange: onChange,
    disabled: disabled,
    style: {
      position: 'absolute',
      opacity: 0,
      width: 0,
      height: 0
    }
  }, rest)), label && /*#__PURE__*/React.createElement("span", null, label));
}
Object.assign(__ds_scope, { Checkbox });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/forms/Checkbox.jsx", error: String((e && e.message) || e) }); }

// components/forms/Input.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/**
 * Text input with optional label, hint, error, and leading icon/adornment.
 */
function Input({
  label,
  hint,
  error,
  leadingIcon,
  id,
  size = 'md',
  style,
  containerStyle,
  disabled,
  ...rest
}) {
  const [focus, setFocus] = React.useState(false);
  const inputId = id || React.useId();
  const pad = size === 'sm' ? '5px 10px' : '8px 12px';
  const fs = size === 'sm' ? '13px' : '14px';
  const borderColor = error ? 'var(--status-fail-solid)' : focus ? 'var(--border-brand)' : 'var(--border-default)';
  return /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      flexDirection: 'column',
      gap: '5px',
      ...containerStyle
    }
  }, label && /*#__PURE__*/React.createElement("label", {
    htmlFor: inputId,
    style: {
      fontSize: '13px',
      fontWeight: 500,
      color: 'var(--text-strong)'
    }
  }, label), /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      alignItems: 'center',
      gap: '8px',
      background: disabled ? 'var(--surface-sunken)' : 'var(--surface-card)',
      border: `1px solid ${borderColor}`,
      borderRadius: 'var(--radius-md)',
      boxShadow: focus ? 'var(--shadow-focus)' : 'none',
      padding: `0 ${pad.split(' ')[1]}`,
      transition: 'border-color var(--dur-fast), box-shadow var(--dur-fast)'
    }
  }, leadingIcon && /*#__PURE__*/React.createElement("span", {
    style: {
      display: 'inline-flex',
      color: 'var(--text-subtle)',
      flex: 'none'
    }
  }, leadingIcon), /*#__PURE__*/React.createElement("input", _extends({
    id: inputId,
    disabled: disabled,
    onFocus: () => setFocus(true),
    onBlur: () => setFocus(false),
    style: {
      flex: 1,
      border: 'none',
      outline: 'none',
      background: 'transparent',
      fontFamily: 'var(--font-sans)',
      fontSize: fs,
      color: 'var(--text-strong)',
      padding: `${pad.split(' ')[0]} 0`,
      minWidth: 0,
      ...style
    }
  }, rest))), (hint || error) && /*#__PURE__*/React.createElement("span", {
    style: {
      fontSize: '12px',
      color: error ? 'var(--status-fail-fg)' : 'var(--text-muted)'
    }
  }, error || hint));
}
Object.assign(__ds_scope, { Input });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/forms/Input.jsx", error: String((e && e.message) || e) }); }

// components/forms/Select.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/**
 * Native select styled to match Input. Pass options as [{value,label}] or children.
 */
function Select({
  label,
  hint,
  options,
  id,
  size = 'md',
  style,
  containerStyle,
  disabled,
  children,
  ...rest
}) {
  const [focus, setFocus] = React.useState(false);
  const selId = id || React.useId();
  const pad = size === 'sm' ? '5px 10px' : '8px 12px';
  const fs = size === 'sm' ? '13px' : '14px';
  return /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      flexDirection: 'column',
      gap: '5px',
      ...containerStyle
    }
  }, label && /*#__PURE__*/React.createElement("label", {
    htmlFor: selId,
    style: {
      fontSize: '13px',
      fontWeight: 500,
      color: 'var(--text-strong)'
    }
  }, label), /*#__PURE__*/React.createElement("div", {
    style: {
      position: 'relative',
      display: 'flex'
    }
  }, /*#__PURE__*/React.createElement("select", _extends({
    id: selId,
    disabled: disabled,
    onFocus: () => setFocus(true),
    onBlur: () => setFocus(false),
    style: {
      appearance: 'none',
      width: '100%',
      fontFamily: 'var(--font-sans)',
      fontSize: fs,
      color: 'var(--text-strong)',
      padding: pad,
      paddingRight: '32px',
      background: disabled ? 'var(--surface-sunken)' : 'var(--surface-card)',
      border: `1px solid ${focus ? 'var(--border-brand)' : 'var(--border-default)'}`,
      borderRadius: 'var(--radius-md)',
      cursor: disabled ? 'not-allowed' : 'pointer',
      boxShadow: focus ? 'var(--shadow-focus)' : 'none',
      outline: 'none',
      transition: 'border-color var(--dur-fast), box-shadow var(--dur-fast)',
      ...style
    }
  }, rest), options ? options.map(o => /*#__PURE__*/React.createElement("option", {
    key: o.value,
    value: o.value
  }, o.label)) : children), /*#__PURE__*/React.createElement("svg", {
    width: "14",
    height: "14",
    viewBox: "0 0 24 24",
    fill: "none",
    stroke: "currentColor",
    strokeWidth: "2",
    strokeLinecap: "round",
    strokeLinejoin: "round",
    style: {
      position: 'absolute',
      right: 10,
      top: '50%',
      transform: 'translateY(-50%)',
      color: 'var(--text-subtle)',
      pointerEvents: 'none'
    }
  }, /*#__PURE__*/React.createElement("polyline", {
    points: "6 9 12 15 18 9"
  }))), hint && /*#__PURE__*/React.createElement("span", {
    style: {
      fontSize: '12px',
      color: 'var(--text-muted)'
    }
  }, hint));
}
Object.assign(__ds_scope, { Select });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/forms/Select.jsx", error: String((e && e.message) || e) }); }

// components/forms/Switch.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/**
 * Toggle switch for on/off settings (e.g. enable a control, mute a finding).
 */
function Switch({
  checked = false,
  onChange,
  label,
  disabled = false,
  id,
  style,
  ...rest
}) {
  const swId = id || React.useId();
  return /*#__PURE__*/React.createElement("label", {
    htmlFor: swId,
    style: {
      display: 'inline-flex',
      alignItems: 'center',
      gap: '10px',
      cursor: disabled ? 'not-allowed' : 'pointer',
      opacity: disabled ? 0.5 : 1,
      fontFamily: 'var(--font-sans)',
      fontSize: '14px',
      color: 'var(--text-body)',
      ...style
    }
  }, /*#__PURE__*/React.createElement("span", {
    onClick: () => !disabled && onChange && onChange(!checked),
    style: {
      width: 34,
      height: 20,
      flex: 'none',
      borderRadius: 'var(--radius-pill)',
      background: checked ? 'var(--brand-500)' : 'var(--slate-300)',
      position: 'relative',
      transition: 'background var(--dur-normal) var(--ease-standard)'
    }
  }, /*#__PURE__*/React.createElement("span", {
    style: {
      position: 'absolute',
      top: 2,
      left: checked ? 16 : 2,
      width: 16,
      height: 16,
      borderRadius: '50%',
      background: '#fff',
      boxShadow: 'var(--shadow-sm)',
      transition: 'left var(--dur-normal) var(--ease-out)'
    }
  })), /*#__PURE__*/React.createElement("input", _extends({
    id: swId,
    type: "checkbox",
    checked: checked,
    disabled: disabled,
    onChange: e => onChange && onChange(e.target.checked),
    style: {
      position: 'absolute',
      opacity: 0,
      width: 0,
      height: 0
    }
  }, rest)), label && /*#__PURE__*/React.createElement("span", null, label));
}
Object.assign(__ds_scope, { Switch });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/forms/Switch.jsx", error: String((e && e.message) || e) }); }

// components/status/Badge.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
const TONES = {
  neutral: {
    color: 'var(--text-body)',
    background: 'var(--surface-sunken)',
    border: 'var(--border-subtle)'
  },
  brand: {
    color: 'var(--brand-700)',
    background: 'var(--brand-50)',
    border: 'var(--brand-100)'
  },
  outline: {
    color: 'var(--text-body)',
    background: 'transparent',
    border: 'var(--border-default)'
  }
};

/**
 * Small neutral label for counts, categories, and metadata.
 */
function Badge({
  tone = 'neutral',
  children,
  style,
  ...rest
}) {
  const t = TONES[tone] || TONES.neutral;
  return /*#__PURE__*/React.createElement("span", _extends({
    style: {
      display: 'inline-flex',
      alignItems: 'center',
      gap: '5px',
      fontFamily: 'var(--font-sans)',
      fontSize: '12px',
      fontWeight: 600,
      lineHeight: 1,
      padding: '3px 8px',
      borderRadius: 'var(--radius-sm)',
      color: t.color,
      background: t.background,
      border: `1px solid ${t.border}`,
      ...style
    }
  }, rest), children);
}
Object.assign(__ds_scope, { Badge });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/status/Badge.jsx", error: String((e && e.message) || e) }); }

// components/status/MetricStat.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/**
 * Big metric / KPI stat block for dashboard headers.
 * Value is mono; optional delta and status accent.
 */
function MetricStat({
  label,
  value,
  unit,
  delta,
  deltaTone = 'neutral',
  status,
  style,
  ...rest
}) {
  const deltaColors = {
    positive: 'var(--status-pass-fg)',
    negative: 'var(--status-fail-fg)',
    neutral: 'var(--text-muted)'
  };
  return /*#__PURE__*/React.createElement("div", _extends({
    style: {
      display: 'flex',
      flexDirection: 'column',
      gap: '6px',
      padding: 'var(--pad-card)',
      background: 'var(--surface-card)',
      border: '1px solid var(--border-subtle)',
      borderRadius: 'var(--radius-lg)',
      boxShadow: 'var(--shadow-xs)',
      borderLeft: status ? `3px solid var(--status-${status}-solid)` : '1px solid var(--border-subtle)',
      ...style
    }
  }, rest), /*#__PURE__*/React.createElement("span", {
    style: {
      fontSize: '11px',
      fontWeight: 600,
      letterSpacing: '0.06em',
      textTransform: 'uppercase',
      color: 'var(--text-muted)'
    }
  }, label), /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      alignItems: 'baseline',
      gap: '4px'
    }
  }, /*#__PURE__*/React.createElement("span", {
    style: {
      fontFamily: 'var(--font-mono)',
      fontSize: '30px',
      fontWeight: 600,
      color: 'var(--text-strong)',
      lineHeight: 1
    }
  }, value), unit && /*#__PURE__*/React.createElement("span", {
    style: {
      fontFamily: 'var(--font-mono)',
      fontSize: '15px',
      color: 'var(--text-muted)'
    }
  }, unit)), delta != null && /*#__PURE__*/React.createElement("span", {
    style: {
      fontSize: '12px',
      fontWeight: 500,
      color: deltaColors[deltaTone]
    }
  }, delta));
}
Object.assign(__ds_scope, { MetricStat });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/status/MetricStat.jsx", error: String((e && e.message) || e) }); }

// components/status/ProgressBar.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/**
 * Coverage / posture bar. Value 0–100. Color follows thresholds unless `status` given.
 */
function ProgressBar({
  value = 0,
  status,
  showLabel = true,
  label,
  height = 8,
  style,
  ...rest
}) {
  const pct = Math.max(0, Math.min(100, value));
  const auto = pct >= 90 ? 'pass' : pct >= 60 ? 'warn' : 'fail';
  const s = status || auto;
  return /*#__PURE__*/React.createElement("div", _extends({
    style: {
      display: 'flex',
      alignItems: 'center',
      gap: '10px',
      ...style
    }
  }, rest), /*#__PURE__*/React.createElement("div", {
    style: {
      flex: 1,
      height,
      background: 'var(--surface-sunken)',
      borderRadius: 'var(--radius-pill)',
      overflow: 'hidden'
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      width: `${pct}%`,
      height: '100%',
      background: `var(--status-${s}-solid)`,
      borderRadius: 'var(--radius-pill)',
      transition: 'width var(--dur-slow) var(--ease-out)'
    }
  })), showLabel && /*#__PURE__*/React.createElement("span", {
    style: {
      fontFamily: 'var(--font-mono)',
      fontSize: '12px',
      fontWeight: 600,
      color: 'var(--text-strong)',
      minWidth: 38,
      textAlign: 'right'
    }
  }, label != null ? label : `${Math.round(pct)}%`));
}
Object.assign(__ds_scope, { ProgressBar });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/status/ProgressBar.jsx", error: String((e && e.message) || e) }); }

// components/status/StatusBadge.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/**
 * Governance status pill: pass / warn / fail / info / none.
 * Status is always carried by color + text (and optional dot), never color alone.
 */
function StatusBadge({
  status = 'none',
  children,
  dot = true,
  solid = false,
  style,
  ...rest
}) {
  const base = {
    display: 'inline-flex',
    alignItems: 'center',
    gap: '6px',
    fontFamily: 'var(--font-sans)',
    fontSize: '12px',
    fontWeight: 600,
    lineHeight: 1,
    padding: '3px 9px',
    borderRadius: 'var(--radius-pill)',
    whiteSpace: 'nowrap'
  };
  const soft = {
    color: `var(--status-${status}-fg)`,
    background: `var(--status-${status}-bg)`,
    border: `1px solid var(--status-${status}-border)`
  };
  const solidStyle = {
    color: status === 'warn' ? 'var(--slate-900)' : '#fff',
    background: `var(--status-${status}-solid)`,
    border: `1px solid var(--status-${status}-solid)`
  };
  return /*#__PURE__*/React.createElement("span", _extends({
    style: {
      ...base,
      ...(solid ? solidStyle : soft),
      ...style
    }
  }, rest), dot && !solid && /*#__PURE__*/React.createElement("span", {
    style: {
      width: 7,
      height: 7,
      borderRadius: '50%',
      background: `var(--status-${status}-solid)`,
      flex: 'none'
    }
  }), children);
}
Object.assign(__ds_scope, { StatusBadge });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/status/StatusBadge.jsx", error: String((e && e.message) || e) }); }

// components/status/Tag.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
/**
 * Removable tag for resource metadata: owners, environments, namespaces, labels.
 * Value is rendered in mono to match resource identifiers.
 */
function Tag({
  label,
  value,
  mono = true,
  onRemove,
  style,
  ...rest
}) {
  return /*#__PURE__*/React.createElement("span", _extends({
    style: {
      display: 'inline-flex',
      alignItems: 'center',
      gap: '6px',
      fontFamily: 'var(--font-sans)',
      fontSize: '12px',
      lineHeight: 1,
      padding: '4px 8px',
      borderRadius: 'var(--radius-sm)',
      background: 'var(--surface-sunken)',
      border: '1px solid var(--border-subtle)',
      color: 'var(--text-body)',
      ...style
    }
  }, rest), label && /*#__PURE__*/React.createElement("span", {
    style: {
      color: 'var(--text-muted)',
      fontWeight: 600
    }
  }, label, ":"), /*#__PURE__*/React.createElement("span", {
    style: {
      fontFamily: mono ? 'var(--font-mono)' : 'inherit',
      color: 'var(--text-strong)'
    }
  }, value), onRemove && /*#__PURE__*/React.createElement("button", {
    type: "button",
    onClick: onRemove,
    "aria-label": "Remove",
    style: {
      border: 'none',
      background: 'none',
      cursor: 'pointer',
      color: 'var(--text-subtle)',
      padding: 0,
      marginLeft: 1,
      fontSize: 14,
      lineHeight: 1
    }
  }, "\xD7"));
}
Object.assign(__ds_scope, { Tag });
})(); } catch (e) { __ds_ns.__errors.push({ path: "components/status/Tag.jsx", error: String((e && e.message) || e) }); }

// ui_kits/posture-scanner/AppShell.jsx
try { (() => {
// App shell: fixed sidebar + topbar, scrollable content.
const {
  Button,
  IconButton,
  StatusBadge,
  Select,
  Input
} = window.OpenMeshGuardDesignSystem_65348c;
const NAV = [{
  id: 'overview',
  label: 'Overview',
  icon: 'dashboard'
}, {
  id: 'findings',
  label: 'Findings',
  icon: 'triangle-alert',
  count: 37
}, {
  id: 'workloads',
  label: 'Workloads',
  icon: 'network'
}, {
  id: 'drift',
  label: 'Drift',
  icon: 'git-compare',
  count: 5
}, {
  id: 'exceptions',
  label: 'Exceptions',
  icon: 'clock'
}, {
  id: 'evidence',
  label: 'Evidence',
  icon: 'file-check'
}];
function NavItem({
  item,
  active,
  onClick
}) {
  const [hover, setHover] = React.useState(false);
  return /*#__PURE__*/React.createElement("button", {
    onClick: onClick,
    onMouseEnter: () => setHover(true),
    onMouseLeave: () => setHover(false),
    style: {
      display: 'flex',
      alignItems: 'center',
      gap: 10,
      width: '100%',
      padding: '8px 10px',
      border: 'none',
      cursor: 'pointer',
      textAlign: 'left',
      borderRadius: 'var(--radius-md)',
      fontFamily: 'var(--font-sans)',
      fontSize: 14,
      fontWeight: active ? 600 : 500,
      color: active ? 'var(--brand-700)' : hover ? 'var(--text-strong)' : 'var(--text-body)',
      background: active ? 'var(--brand-50)' : hover ? 'var(--surface-hover)' : 'transparent',
      transition: 'background var(--dur-fast), color var(--dur-fast)'
    }
  }, /*#__PURE__*/React.createElement(Icon, {
    name: item.icon,
    size: 17,
    style: {
      color: active ? 'var(--brand-600)' : 'var(--text-muted)'
    }
  }), /*#__PURE__*/React.createElement("span", {
    style: {
      flex: 1
    }
  }, item.label), item.count != null && /*#__PURE__*/React.createElement("span", {
    style: {
      fontFamily: 'var(--font-mono)',
      fontSize: 11,
      fontWeight: 600,
      color: active ? 'var(--brand-700)' : 'var(--text-muted)'
    }
  }, item.count));
}
function AppShell({
  active,
  onNav,
  cluster,
  onCluster,
  onScan,
  children,
  title,
  subtitle,
  actions
}) {
  return /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      height: '100%',
      background: 'var(--surface-page)',
      color: 'var(--text-body)',
      fontFamily: 'var(--font-sans)'
    }
  }, /*#__PURE__*/React.createElement("aside", {
    style: {
      width: 'var(--sidebar-width)',
      flex: 'none',
      background: 'var(--surface-card)',
      borderRight: '1px solid var(--border-subtle)',
      display: 'flex',
      flexDirection: 'column'
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      height: 'var(--topbar-height)',
      display: 'flex',
      alignItems: 'center',
      padding: '0 16px',
      borderBottom: '1px solid var(--border-subtle)'
    }
  }, /*#__PURE__*/React.createElement("img", {
    src: "../../assets/logo-wordmark.svg",
    height: "26",
    alt: "OpenMeshGuard"
  })), /*#__PURE__*/React.createElement("nav", {
    style: {
      padding: 12,
      display: 'flex',
      flexDirection: 'column',
      gap: 2,
      flex: 1
    }
  }, /*#__PURE__*/React.createElement("div", {
    className: "omg-eyebrow",
    style: {
      padding: '8px 10px 4px'
    }
  }, "Posture"), NAV.map(it => /*#__PURE__*/React.createElement(NavItem, {
    key: it.id,
    item: it,
    active: active === it.id,
    onClick: () => onNav(it.id)
  }))), /*#__PURE__*/React.createElement("div", {
    style: {
      padding: 12,
      borderTop: '1px solid var(--border-subtle)'
    }
  }, /*#__PURE__*/React.createElement(NavItem, {
    item: {
      id: 'settings',
      label: 'Settings',
      icon: 'settings'
    },
    active: active === 'settings',
    onClick: () => onNav('settings')
  }))), /*#__PURE__*/React.createElement("div", {
    style: {
      flex: 1,
      minWidth: 0,
      display: 'flex',
      flexDirection: 'column'
    }
  }, /*#__PURE__*/React.createElement("header", {
    style: {
      height: 'var(--topbar-height)',
      flex: 'none',
      display: 'flex',
      alignItems: 'center',
      gap: 12,
      padding: '0 24px',
      background: 'var(--surface-card)',
      borderBottom: '1px solid var(--border-subtle)'
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      alignItems: 'center',
      gap: 8,
      color: 'var(--text-muted)',
      fontSize: 13
    }
  }, /*#__PURE__*/React.createElement(Icon, {
    name: "box",
    size: 15
  }), /*#__PURE__*/React.createElement("span", null, "Cluster")), /*#__PURE__*/React.createElement("div", {
    style: {
      width: 150
    }
  }, /*#__PURE__*/React.createElement(Select, {
    value: cluster,
    onChange: e => onCluster(e.target.value),
    size: "sm",
    options: window.OMG_DATA.clusters.map(c => ({
      value: c,
      label: c
    }))
  })), /*#__PURE__*/React.createElement("div", {
    style: {
      flex: 1,
      maxWidth: 320,
      marginLeft: 8
    }
  }, /*#__PURE__*/React.createElement(Input, {
    size: "sm",
    placeholder: "Search resources, owners, namespaces",
    leadingIcon: /*#__PURE__*/React.createElement(Icon, {
      name: "search",
      size: 15
    })
  })), /*#__PURE__*/React.createElement("div", {
    style: {
      flex: 1
    }
  }), /*#__PURE__*/React.createElement("span", {
    style: {
      display: 'inline-flex',
      alignItems: 'center',
      gap: 6,
      fontSize: 12,
      color: 'var(--text-muted)'
    }
  }, /*#__PURE__*/React.createElement(Icon, {
    name: "refresh",
    size: 14
  }), " Last scan 4h ago"), /*#__PURE__*/React.createElement(IconButton, {
    title: "Notifications",
    variant: "ghost"
  }, /*#__PURE__*/React.createElement(Icon, {
    name: "bell",
    size: 18
  })), /*#__PURE__*/React.createElement(Button, {
    size: "sm",
    variant: "primary",
    leftIcon: /*#__PURE__*/React.createElement(Icon, {
      name: "refresh",
      size: 15
    }),
    onClick: onScan
  }, "Run scan")), /*#__PURE__*/React.createElement("main", {
    style: {
      flex: 1,
      overflowY: 'auto',
      padding: '24px'
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      maxWidth: 'var(--content-max)',
      margin: '0 auto'
    }
  }, (title || actions) && /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      alignItems: 'flex-start',
      justifyContent: 'space-between',
      gap: 16,
      marginBottom: 20
    }
  }, /*#__PURE__*/React.createElement("div", null, title && /*#__PURE__*/React.createElement("h1", {
    style: {
      fontSize: 24,
      fontWeight: 600,
      color: 'var(--text-strong)'
    }
  }, title), subtitle && /*#__PURE__*/React.createElement("p", {
    style: {
      fontSize: 14,
      color: 'var(--text-muted)',
      marginTop: 4
    }
  }, subtitle)), actions && /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      gap: 8,
      flex: 'none'
    }
  }, actions)), children))));
}
window.AppShell = AppShell;
})(); } catch (e) { __ds_ns.__errors.push({ path: "ui_kits/posture-scanner/AppShell.jsx", error: String((e && e.message) || e) }); }

// ui_kits/posture-scanner/Evidence.jsx
try { (() => {
// Evidence / report export screen.
const {
  Card: ECard,
  Button: EButton,
  Checkbox: ECheckbox,
  StatusBadge: EStatus,
  Badge: EBadge,
  Select: ESelect
} = window.OpenMeshGuardDesignSystem_65348c;
function Evidence() {
  const d = window.OMG_DATA;
  const [sections, setSections] = React.useState({
    summary: true,
    mtls: true,
    authz: true,
    exposure: true,
    ownership: false,
    exceptions: true,
    drift: false
  });
  const toggle = k => setSections(s => ({
    ...s,
    [k]: !s[k]
  }));
  const opts = [['summary', 'Executive summary'], ['mtls', 'mTLS enforcement coverage'], ['authz', 'Authorization policy coverage'], ['exposure', 'Exposure & ingress risk'], ['ownership', 'Ownership & metadata gaps'], ['exceptions', 'Active & expiring exceptions'], ['drift', 'GitOps drift log']];
  const selected = Object.values(sections).filter(Boolean).length;
  return /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'grid',
      gridTemplateColumns: '1fr 340px',
      gap: 20,
      alignItems: 'start'
    }
  }, /*#__PURE__*/React.createElement(ECard, {
    title: "Build evidence report",
    subtitle: "Audit-ready export \u2014 no spreadsheets required"
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      gap: 12,
      marginBottom: 20
    }
  }, /*#__PURE__*/React.createElement(ESelect, {
    label: "Scope",
    containerStyle: {
      flex: 1
    },
    options: d.clusters.map(c => ({
      value: c,
      label: c
    })).concat([{
      value: 'all',
      label: 'All clusters'
    }])
  }), /*#__PURE__*/React.createElement(ESelect, {
    label: "Framework",
    containerStyle: {
      flex: 1
    },
    options: [{
      value: 'soc2',
      label: 'SOC 2'
    }, {
      value: 'iso',
      label: 'ISO 27001'
    }, {
      value: 'pci',
      label: 'PCI DSS'
    }, {
      value: 'none',
      label: 'None (raw evidence)'
    }]
  }), /*#__PURE__*/React.createElement(ESelect, {
    label: "Format",
    containerStyle: {
      width: 130
    },
    options: [{
      value: 'pdf',
      label: 'PDF'
    }, {
      value: 'csv',
      label: 'CSV'
    }, {
      value: 'json',
      label: 'JSON'
    }]
  })), /*#__PURE__*/React.createElement("div", {
    className: "omg-eyebrow",
    style: {
      marginBottom: 12
    }
  }, "Sections"), /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      flexDirection: 'column',
      gap: 12
    }
  }, opts.map(([k, label]) => /*#__PURE__*/React.createElement("div", {
    key: k,
    style: {
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'space-between',
      paddingBottom: 12,
      borderBottom: '1px solid var(--border-subtle)'
    }
  }, /*#__PURE__*/React.createElement(ECheckbox, {
    checked: sections[k],
    onChange: () => toggle(k),
    label: label
  }), /*#__PURE__*/React.createElement(EStatus, {
    status: k === 'ownership' || k === 'drift' ? 'warn' : 'pass'
  }, k === 'ownership' || k === 'drift' ? 'Findings present' : 'Ready'))))), /*#__PURE__*/React.createElement(ECard, {
    title: "Report summary"
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      flexDirection: 'column',
      gap: 14
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      justifyContent: 'space-between',
      fontSize: 13
    }
  }, /*#__PURE__*/React.createElement("span", {
    style: {
      color: 'var(--text-muted)'
    }
  }, "Scope"), /*#__PURE__*/React.createElement("code", {
    style: {
      fontFamily: 'var(--font-mono)'
    }
  }, "prod-eu-1")), /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      justifyContent: 'space-between',
      fontSize: 13
    }
  }, /*#__PURE__*/React.createElement("span", {
    style: {
      color: 'var(--text-muted)'
    }
  }, "Sections"), /*#__PURE__*/React.createElement("span", {
    style: {
      fontWeight: 600,
      color: 'var(--text-strong)'
    }
  }, selected, " of ", opts.length)), /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      justifyContent: 'space-between',
      fontSize: 13
    }
  }, /*#__PURE__*/React.createElement("span", {
    style: {
      color: 'var(--text-muted)'
    }
  }, "Findings included"), /*#__PURE__*/React.createElement("span", {
    style: {
      fontWeight: 600,
      color: 'var(--text-strong)'
    }
  }, d.metrics.openFindings)), /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      justifyContent: 'space-between',
      fontSize: 13
    }
  }, /*#__PURE__*/React.createElement("span", {
    style: {
      color: 'var(--text-muted)'
    }
  }, "Generated by"), /*#__PURE__*/React.createElement("span", null, "OpenMeshGuard v1.24")), /*#__PURE__*/React.createElement("div", {
    style: {
      height: 1,
      background: 'var(--border-subtle)'
    }
  }), /*#__PURE__*/React.createElement(EButton, {
    variant: "primary",
    fullWidth: true,
    leftIcon: /*#__PURE__*/React.createElement(Icon, {
      name: "file-check",
      size: 16
    })
  }, "Generate evidence (PDF)"), /*#__PURE__*/React.createElement(EButton, {
    variant: "secondary",
    fullWidth: true,
    leftIcon: /*#__PURE__*/React.createElement(Icon, {
      name: "clock",
      size: 15
    })
  }, "Schedule monthly export"), /*#__PURE__*/React.createElement("p", {
    style: {
      fontSize: 12,
      color: 'var(--text-muted)',
      textAlign: 'center',
      margin: 0
    }
  }, "Signed & timestamped. Retained for 12 months."))));
}
window.Evidence = Evidence;
})(); } catch (e) { __ds_ns.__errors.push({ path: "ui_kits/posture-scanner/Evidence.jsx", error: String((e && e.message) || e) }); }

// ui_kits/posture-scanner/Findings.jsx
try { (() => {
// Findings list screen with filter tabs and a data table.
const {
  Card: FCard,
  DataTable: FTable,
  Tabs: FTabs,
  StatusBadge: FStatus,
  Badge: FBadge,
  Avatar: FAvatar,
  Button: FButton,
  Input: FInput
} = window.OpenMeshGuardDesignSystem_65348c;
function Findings({
  onOpenResource
}) {
  const d = window.OMG_DATA;
  const [tab, setTab] = React.useState('all');
  const controls = ['all', 'mTLS', 'Authorization', 'Exposure', 'Drift', 'Ownership'];
  const rows = tab === 'all' ? d.findings : d.findings.filter(f => f.control === tab);
  return /*#__PURE__*/React.createElement(FCard, {
    padded: false,
    title: "Findings",
    subtitle: `${rows.length} of ${d.findings.length} findings`,
    actions: /*#__PURE__*/React.createElement(FButton, {
      size: "sm",
      variant: "secondary",
      leftIcon: /*#__PURE__*/React.createElement(Icon, {
        name: "download",
        size: 15
      })
    }, "Export evidence")
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      padding: '0 20px'
    }
  }, /*#__PURE__*/React.createElement(FTabs, {
    value: tab,
    onChange: setTab,
    items: controls.map(c => ({
      id: c,
      label: c === 'all' ? 'All' : c,
      count: c === 'all' ? d.findings.length : d.findings.filter(f => f.control === c).length || undefined
    }))
  })), /*#__PURE__*/React.createElement(FTable, {
    rowKey: "id",
    onRowClick: onOpenResource,
    columns: [{
      key: 'severity',
      header: 'Severity',
      width: 110,
      render: (v, r) => /*#__PURE__*/React.createElement(FStatus, {
        status: v,
        solid: v === 'fail'
      }, r.sevLabel)
    }, {
      key: 'title',
      header: 'Finding',
      render: (v, r) => /*#__PURE__*/React.createElement("div", {
        style: {
          display: 'flex',
          flexDirection: 'column',
          gap: 2
        }
      }, /*#__PURE__*/React.createElement("span", {
        style: {
          color: 'var(--text-strong)',
          fontFamily: 'var(--font-sans)',
          fontWeight: 500,
          whiteSpace: 'normal'
        }
      }, v), /*#__PURE__*/React.createElement("code", {
        style: {
          fontFamily: 'var(--font-mono)',
          fontSize: 11.5,
          color: 'var(--text-muted)'
        }
      }, r.id))
    }, {
      key: 'kind',
      header: 'Kind',
      render: v => /*#__PURE__*/React.createElement("code", {
        style: {
          fontFamily: 'var(--font-mono)',
          fontSize: 12,
          color: 'var(--text-body)'
        }
      }, v)
    }, {
      key: 'resource',
      header: 'Resource',
      mono: true
    }, {
      key: 'owner',
      header: 'Owner',
      render: v => v === 'unowned' ? /*#__PURE__*/React.createElement(FStatus, {
        status: "warn"
      }, "Unowned") : /*#__PURE__*/React.createElement("span", {
        style: {
          display: 'inline-flex',
          alignItems: 'center',
          gap: 6
        }
      }, /*#__PURE__*/React.createElement(FAvatar, {
        name: v,
        size: "sm"
      }), /*#__PURE__*/React.createElement("span", {
        style: {
          fontFamily: 'var(--font-mono)',
          fontSize: 12
        }
      }, v))
    }, {
      key: 'control',
      header: 'Control',
      render: v => /*#__PURE__*/React.createElement(FBadge, {
        tone: "neutral"
      }, v)
    }, {
      key: 'age',
      header: 'Age',
      align: 'right',
      mono: true
    }],
    rows: rows
  }));
}
window.Findings = Findings;
})(); } catch (e) { __ds_ns.__errors.push({ path: "ui_kits/posture-scanner/Findings.jsx", error: String((e && e.message) || e) }); }

// ui_kits/posture-scanner/Icons.jsx
try { (() => {
function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
// Lucide-style outline icons (1.5–2px stroke, currentColor). Substitution for a source icon set.
// Usage: <Icon name="shield-check" size={18} />
const ICON_PATHS = {
  'shield-check': /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement("path", {
    d: "M20 13c0 5-3.5 7.5-7.66 8.95a1 1 0 0 1-.67-.01C7.5 20.5 4 18 4 13V6a1 1 0 0 1 1-1c2 0 4.5-1.2 6.24-2.72a1.17 1.17 0 0 1 1.52 0C14.51 3.81 17 5 19 5a1 1 0 0 1 1 1z"
  }), /*#__PURE__*/React.createElement("path", {
    d: "m9 12 2 2 4-4"
  })),
  'shield-alert': /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement("path", {
    d: "M20 13c0 5-3.5 7.5-7.66 8.95a1 1 0 0 1-.67-.01C7.5 20.5 4 18 4 13V6a1 1 0 0 1 1-1c2 0 4.5-1.2 6.24-2.72a1.17 1.17 0 0 1 1.52 0C14.51 3.81 17 5 19 5a1 1 0 0 1 1 1z"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M12 8v4"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M12 16h.01"
  })),
  'triangle-alert': /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement("path", {
    d: "m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M12 9v4"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M12 17h.01"
  })),
  'circle-help': /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement("circle", {
    cx: "12",
    cy: "12",
    r: "10"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M12 17h.01"
  })),
  'dashboard': /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement("rect", {
    width: "7",
    height: "9",
    x: "3",
    y: "3",
    rx: "1"
  }), /*#__PURE__*/React.createElement("rect", {
    width: "7",
    height: "5",
    x: "14",
    y: "3",
    rx: "1"
  }), /*#__PURE__*/React.createElement("rect", {
    width: "7",
    height: "9",
    x: "14",
    y: "12",
    rx: "1"
  }), /*#__PURE__*/React.createElement("rect", {
    width: "7",
    height: "5",
    x: "3",
    y: "16",
    rx: "1"
  })),
  'list': /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement("path", {
    d: "M8 6h13"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M8 12h13"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M8 18h13"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M3 6h.01"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M3 12h.01"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M3 18h.01"
  })),
  'git-compare': /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement("circle", {
    cx: "18",
    cy: "18",
    r: "3"
  }), /*#__PURE__*/React.createElement("circle", {
    cx: "6",
    cy: "6",
    r: "3"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M13 6h3a2 2 0 0 1 2 2v7"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M11 18H8a2 2 0 0 1-2-2V9"
  })),
  'file-check': /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement("path", {
    d: "M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M14 2v4a2 2 0 0 0 2 2h4"
  }), /*#__PURE__*/React.createElement("path", {
    d: "m9 15 2 2 4-4"
  })),
  'user': /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement("circle", {
    cx: "12",
    cy: "8",
    r: "5"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M20 21a8 8 0 0 0-16 0"
  })),
  'clock': /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement("circle", {
    cx: "12",
    cy: "12",
    r: "10"
  }), /*#__PURE__*/React.createElement("polyline", {
    points: "12 6 12 12 16 14"
  })),
  'search': /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement("circle", {
    cx: "11",
    cy: "11",
    r: "8"
  }), /*#__PURE__*/React.createElement("path", {
    d: "m21 21-4.3-4.3"
  })),
  'download': /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement("path", {
    d: "M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"
  }), /*#__PURE__*/React.createElement("polyline", {
    points: "7 10 12 15 17 10"
  }), /*#__PURE__*/React.createElement("line", {
    x1: "12",
    y1: "15",
    x2: "12",
    y2: "3"
  })),
  'chevron-right': /*#__PURE__*/React.createElement("polyline", {
    points: "9 18 15 12 9 6"
  }),
  'chevron-down': /*#__PURE__*/React.createElement("polyline", {
    points: "6 9 12 15 18 9"
  }),
  'filter': /*#__PURE__*/React.createElement("polygon", {
    points: "22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"
  }),
  'settings': /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement("path", {
    d: "M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"
  }), /*#__PURE__*/React.createElement("circle", {
    cx: "12",
    cy: "12",
    r: "3"
  })),
  'bell': /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement("path", {
    d: "M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M10.3 21a1.94 1.94 0 0 0 3.4 0"
  })),
  'plus': /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement("path", {
    d: "M5 12h14"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M12 5v14"
  })),
  'external-link': /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement("path", {
    d: "M15 3h6v6"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M10 14 21 3"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"
  })),
  'lock': /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement("rect", {
    width: "18",
    height: "11",
    x: "3",
    y: "11",
    rx: "2",
    ry: "2"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M7 11V7a5 5 0 0 1 10 0v4"
  })),
  'globe': /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement("circle", {
    cx: "12",
    cy: "12",
    r: "10"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M2 12h20"
  })),
  'box': /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement("path", {
    d: "M21 8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16Z"
  }), /*#__PURE__*/React.createElement("path", {
    d: "m3.3 7 8.7 5 8.7-5"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M12 22V12"
  })),
  'network': /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement("rect", {
    x: "16",
    y: "16",
    width: "6",
    height: "6",
    rx: "1"
  }), /*#__PURE__*/React.createElement("rect", {
    x: "2",
    y: "16",
    width: "6",
    height: "6",
    rx: "1"
  }), /*#__PURE__*/React.createElement("rect", {
    x: "9",
    y: "2",
    width: "6",
    height: "6",
    rx: "1"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M5 16v-3a1 1 0 0 1 1-1h12a1 1 0 0 1 1 1v3"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M12 12V8"
  })),
  'refresh': /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement("path", {
    d: "M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M21 3v5h-5"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M8 16H3v5"
  })),
  'check': /*#__PURE__*/React.createElement("polyline", {
    points: "20 6 9 17 4 12"
  }),
  'x': /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement("path", {
    d: "M18 6 6 18"
  }), /*#__PURE__*/React.createElement("path", {
    d: "m6 6 12 12"
  })),
  'arrow-left': /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement("path", {
    d: "m12 19-7-7 7-7"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M19 12H5"
  })),
  'file-text': /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement("path", {
    d: "M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M14 2v4a2 2 0 0 0 2 2h4"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M16 13H8"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M16 17H8"
  }), /*#__PURE__*/React.createElement("path", {
    d: "M10 9H8"
  }))
};
function Icon({
  name,
  size = 18,
  strokeWidth = 2,
  style,
  ...rest
}) {
  return /*#__PURE__*/React.createElement("svg", _extends({
    width: size,
    height: size,
    viewBox: "0 0 24 24",
    fill: "none",
    stroke: "currentColor",
    strokeWidth: strokeWidth,
    strokeLinecap: "round",
    strokeLinejoin: "round",
    style: {
      display: 'inline-block',
      flex: 'none',
      ...style
    }
  }, rest), ICON_PATHS[name] || null);
}
window.Icon = Icon;
})(); } catch (e) { __ds_ns.__errors.push({ path: "ui_kits/posture-scanner/Icons.jsx", error: String((e && e.message) || e) }); }

// ui_kits/posture-scanner/Overview.jsx
try { (() => {
// Overview dashboard screen.
const {
  Card: OvCard,
  MetricStat,
  ProgressBar: OvProgress,
  StatusBadge: OvStatus,
  Button: OvButton
} = window.OpenMeshGuardDesignSystem_65348c;
function SeverityBar({
  items
}) {
  const total = items.reduce((a, b) => a + b.count, 0);
  return /*#__PURE__*/React.createElement("div", null, /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      height: 10,
      borderRadius: 'var(--radius-pill)',
      overflow: 'hidden',
      marginBottom: 14
    }
  }, items.map(s => /*#__PURE__*/React.createElement("div", {
    key: s.label,
    title: `${s.label}: ${s.count}`,
    style: {
      width: `${s.count / total * 100}%`,
      background: `var(--status-${s.status}-solid)`
    }
  }))), /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      flexDirection: 'column',
      gap: 10
    }
  }, items.map(s => /*#__PURE__*/React.createElement("div", {
    key: s.label,
    style: {
      display: 'flex',
      alignItems: 'center',
      gap: 10,
      fontSize: 13
    }
  }, /*#__PURE__*/React.createElement("span", {
    style: {
      width: 8,
      height: 8,
      borderRadius: '50%',
      background: `var(--status-${s.status}-solid)`,
      flex: 'none'
    }
  }), /*#__PURE__*/React.createElement("span", {
    style: {
      flex: 1,
      color: 'var(--text-body)'
    }
  }, s.label), /*#__PURE__*/React.createElement("span", {
    style: {
      fontFamily: 'var(--font-mono)',
      fontWeight: 600,
      color: 'var(--text-strong)'
    }
  }, s.count)))));
}
function ControlRow({
  c,
  onOpen
}) {
  const [hover, setHover] = React.useState(false);
  return /*#__PURE__*/React.createElement("div", {
    onClick: onOpen,
    onMouseEnter: () => setHover(true),
    onMouseLeave: () => setHover(false),
    style: {
      display: 'flex',
      alignItems: 'center',
      gap: 16,
      padding: '12px 20px',
      borderBottom: '1px solid var(--border-subtle)',
      cursor: 'pointer',
      background: hover ? 'var(--surface-hover)' : 'transparent'
    }
  }, /*#__PURE__*/React.createElement(OvStatus, {
    status: c.status,
    dot: true
  }), /*#__PURE__*/React.createElement("span", {
    style: {
      flex: 1,
      fontSize: 14,
      color: 'var(--text-strong)',
      fontWeight: 500
    }
  }, c.name), /*#__PURE__*/React.createElement("div", {
    style: {
      width: 200
    }
  }, /*#__PURE__*/React.createElement(OvProgress, {
    value: c.coverage
  })));
}
function Overview({
  onOpenFindings,
  onOpenResource
}) {
  const d = window.OMG_DATA;
  return /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      flexDirection: 'column',
      gap: 20
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'grid',
      gridTemplateColumns: 'repeat(4, 1fr)',
      gap: 12
    }
  }, /*#__PURE__*/React.createElement(MetricStat, {
    label: "mTLS coverage",
    value: d.metrics.mtlsCoverage,
    unit: "%",
    delta: "+4 since last scan",
    deltaTone: "positive",
    status: "warn"
  }), /*#__PURE__*/React.createElement(MetricStat, {
    label: "Open findings",
    value: d.metrics.openFindings,
    delta: `${d.metrics.critical} critical`,
    deltaTone: "negative",
    status: "fail"
  }), /*#__PURE__*/React.createElement(MetricStat, {
    label: "Owned resources",
    value: d.metrics.ownedPct,
    unit: "%",
    delta: "stable",
    status: "pass"
  }), /*#__PURE__*/React.createElement(MetricStat, {
    label: "Active exceptions",
    value: d.metrics.exceptions,
    delta: "2 expiring soon",
    deltaTone: "neutral",
    status: "warn"
  })), /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'grid',
      gridTemplateColumns: '1.6fr 1fr',
      gap: 20,
      alignItems: 'start'
    }
  }, /*#__PURE__*/React.createElement(OvCard, {
    padded: false,
    title: "Control posture",
    subtitle: "Coverage across enterprise controls",
    actions: /*#__PURE__*/React.createElement(OvButton, {
      size: "sm",
      variant: "secondary",
      onClick: onOpenFindings
    }, "View all findings")
  }, d.controls.map(c => /*#__PURE__*/React.createElement(ControlRow, {
    key: c.name,
    c: c,
    onOpen: onOpenFindings
  }))), /*#__PURE__*/React.createElement(OvCard, {
    title: "Findings by severity",
    subtitle: `${d.metrics.openFindings} open`
  }, /*#__PURE__*/React.createElement(SeverityBar, {
    items: d.severityBreakdown
  }))), /*#__PURE__*/React.createElement(OvCard, {
    padded: false,
    title: "Recent critical findings",
    actions: /*#__PURE__*/React.createElement(OvButton, {
      size: "sm",
      variant: "ghost",
      onClick: onOpenFindings
    }, "Open findings")
  }, d.findings.filter(f => f.sevLabel === 'Critical' || f.sevLabel === 'High').slice(0, 4).map(f => /*#__PURE__*/React.createElement("div", {
    key: f.id,
    onClick: () => onOpenResource(f),
    style: {
      display: 'flex',
      alignItems: 'center',
      gap: 14,
      padding: '12px 20px',
      borderBottom: '1px solid var(--border-subtle)',
      cursor: 'pointer'
    }
  }, /*#__PURE__*/React.createElement(OvStatus, {
    status: f.severity,
    solid: f.severity === 'fail'
  }, f.sevLabel), /*#__PURE__*/React.createElement("span", {
    style: {
      flex: 1,
      fontSize: 14,
      color: 'var(--text-strong)'
    }
  }, f.title), /*#__PURE__*/React.createElement("code", {
    style: {
      fontFamily: 'var(--font-mono)',
      fontSize: 12,
      color: 'var(--text-muted)'
    }
  }, f.ns, "/", f.resource), /*#__PURE__*/React.createElement(Icon, {
    name: "chevron-right",
    size: 16,
    style: {
      color: 'var(--text-subtle)'
    }
  })))));
}
window.Overview = Overview;
})(); } catch (e) { __ds_ns.__errors.push({ path: "ui_kits/posture-scanner/Overview.jsx", error: String((e && e.message) || e) }); }

// ui_kits/posture-scanner/ResourceDetail.jsx
try { (() => {
// Resource / finding detail drill-in.
const {
  Card: RCard,
  StatusBadge: RStatus,
  Tag: RTag,
  Button: RButton,
  Avatar: RAvatar,
  ProgressBar: RProgress
} = window.OpenMeshGuardDesignSystem_65348c;
function Row({
  label,
  children
}) {
  return /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      gap: 16,
      padding: '10px 0',
      borderBottom: '1px solid var(--border-subtle)'
    }
  }, /*#__PURE__*/React.createElement("span", {
    style: {
      width: 150,
      flex: 'none',
      fontSize: 13,
      color: 'var(--text-muted)'
    }
  }, label), /*#__PURE__*/React.createElement("div", {
    style: {
      flex: 1,
      fontSize: 13,
      color: 'var(--text-body)'
    }
  }, children));
}
function ResourceDetail({
  finding,
  onBack
}) {
  const f = finding || window.OMG_DATA.findings[0];
  const yaml = `apiVersion: security.istio.io/v1
kind: PeerAuthentication
metadata:
  name: default
  namespace: ${f.ns}
spec:
  mtls:
    mode: PERMISSIVE   # expected: STRICT`;
  const checks = [{
    name: 'Strict mTLS enforced',
    status: f.control === 'mTLS' ? 'fail' : 'pass'
  }, {
    name: 'AuthorizationPolicy present',
    status: f.control === 'Authorization' ? 'fail' : 'pass'
  }, {
    name: 'Not publicly exposed',
    status: f.control === 'Exposure' ? 'fail' : 'pass'
  }, {
    name: 'Ownership metadata complete',
    status: f.owner === 'unowned' ? 'fail' : 'pass'
  }, {
    name: 'In sync with GitOps',
    status: f.control === 'Drift' ? 'fail' : 'pass'
  }];
  return /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      flexDirection: 'column',
      gap: 20
    }
  }, /*#__PURE__*/React.createElement("button", {
    onClick: onBack,
    style: {
      display: 'inline-flex',
      alignItems: 'center',
      gap: 6,
      background: 'none',
      border: 'none',
      cursor: 'pointer',
      color: 'var(--text-muted)',
      fontSize: 13,
      padding: 0,
      width: 'fit-content',
      fontFamily: 'var(--font-sans)'
    }
  }, /*#__PURE__*/React.createElement(Icon, {
    name: "arrow-left",
    size: 15
  }), " Back to findings"), /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      alignItems: 'flex-start',
      justifyContent: 'space-between',
      gap: 16
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      flexDirection: 'column',
      gap: 8
    }
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      alignItems: 'center',
      gap: 10
    }
  }, /*#__PURE__*/React.createElement(RStatus, {
    status: f.severity,
    solid: f.severity === 'fail'
  }, f.sevLabel), /*#__PURE__*/React.createElement("code", {
    style: {
      fontFamily: 'var(--font-mono)',
      fontSize: 12,
      color: 'var(--text-muted)'
    }
  }, f.id)), /*#__PURE__*/React.createElement("h1", {
    style: {
      fontSize: 22,
      fontWeight: 600,
      color: 'var(--text-strong)',
      maxWidth: 720
    }
  }, f.title), /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      gap: 8,
      flexWrap: 'wrap'
    }
  }, /*#__PURE__*/React.createElement(RTag, {
    label: "kind",
    value: f.kind
  }), /*#__PURE__*/React.createElement(RTag, {
    label: "resource",
    value: `${f.ns}/${f.resource}`
  }), /*#__PURE__*/React.createElement(RTag, {
    label: "control",
    value: f.control
  }))), /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      gap: 8,
      flex: 'none'
    }
  }, /*#__PURE__*/React.createElement(RButton, {
    variant: "secondary",
    leftIcon: /*#__PURE__*/React.createElement(Icon, {
      name: "user",
      size: 15
    })
  }, "Assign owner"), /*#__PURE__*/React.createElement(RButton, {
    variant: "secondary",
    leftIcon: /*#__PURE__*/React.createElement(Icon, {
      name: "clock",
      size: 15
    })
  }, "Request exception"), /*#__PURE__*/React.createElement(RButton, {
    variant: "primary",
    leftIcon: /*#__PURE__*/React.createElement(Icon, {
      name: "download",
      size: 15
    })
  }, "Export evidence"))), /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'grid',
      gridTemplateColumns: '1fr 1fr',
      gap: 20,
      alignItems: 'start'
    }
  }, /*#__PURE__*/React.createElement(RCard, {
    title: "Control posture",
    subtitle: "Evaluated against enterprise controls"
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      flexDirection: 'column',
      gap: 12
    }
  }, checks.map(c => /*#__PURE__*/React.createElement("div", {
    key: c.name,
    style: {
      display: 'flex',
      alignItems: 'center',
      gap: 10
    }
  }, /*#__PURE__*/React.createElement(Icon, {
    name: c.status === 'pass' ? 'shield-check' : 'shield-alert',
    size: 18,
    style: {
      color: `var(--status-${c.status}-solid)`
    }
  }), /*#__PURE__*/React.createElement("span", {
    style: {
      flex: 1,
      fontSize: 14,
      color: 'var(--text-strong)'
    }
  }, c.name), /*#__PURE__*/React.createElement(RStatus, {
    status: c.status
  }, c.status === 'pass' ? 'Pass' : 'Fail'))))), /*#__PURE__*/React.createElement(RCard, {
    title: "Ownership & metadata"
  }, /*#__PURE__*/React.createElement(Row, {
    label: "Owner"
  }, f.owner === 'unowned' ? /*#__PURE__*/React.createElement(RStatus, {
    status: "warn"
  }, "Unowned") : /*#__PURE__*/React.createElement("span", {
    style: {
      display: 'inline-flex',
      alignItems: 'center',
      gap: 6
    }
  }, /*#__PURE__*/React.createElement(RAvatar, {
    name: f.owner,
    size: "sm"
  }), /*#__PURE__*/React.createElement("code", {
    style: {
      fontFamily: 'var(--font-mono)',
      fontSize: 12
    }
  }, f.owner))), /*#__PURE__*/React.createElement(Row, {
    label: "Namespace"
  }, /*#__PURE__*/React.createElement("code", {
    style: {
      fontFamily: 'var(--font-mono)'
    }
  }, f.ns)), /*#__PURE__*/React.createElement(Row, {
    label: "Environment"
  }, /*#__PURE__*/React.createElement("code", {
    style: {
      fontFamily: 'var(--font-mono)'
    }
  }, "production")), /*#__PURE__*/React.createElement(Row, {
    label: "Repository"
  }, f.owner === 'unowned' ? /*#__PURE__*/React.createElement(RStatus, {
    status: "warn"
  }, "Missing") : /*#__PURE__*/React.createElement("a", {
    href: "#"
  }, "git.corp/mesh/", f.resource)), /*#__PURE__*/React.createElement(Row, {
    label: "First detected"
  }, f.age, " ago"))), /*#__PURE__*/React.createElement(RCard, {
    title: "Evidence",
    subtitle: "Observed in-cluster configuration",
    actions: /*#__PURE__*/React.createElement(RButton, {
      size: "sm",
      variant: "ghost",
      leftIcon: /*#__PURE__*/React.createElement(Icon, {
        name: "external-link",
        size: 14
      })
    }, "Open in cluster")
  }, /*#__PURE__*/React.createElement("pre", {
    style: {
      margin: 0,
      fontFamily: 'var(--font-mono)',
      fontSize: 13,
      lineHeight: 1.6,
      color: 'var(--slate-100)',
      background: 'var(--slate-900)',
      padding: 16,
      borderRadius: 'var(--radius-md)',
      overflowX: 'auto'
    }
  }, yaml)));
}
window.ResourceDetail = ResourceDetail;
})(); } catch (e) { __ds_ns.__errors.push({ path: "ui_kits/posture-scanner/ResourceDetail.jsx", error: String((e && e.message) || e) }); }

// ui_kits/posture-scanner/Workloads.jsx
try { (() => {
// Workloads inventory screen.
const {
  Card: WCard,
  DataTable: WTable,
  StatusBadge: WStatus,
  Avatar: WAvatar,
  ProgressBar: WProgress,
  Button: WButton
} = window.OpenMeshGuardDesignSystem_65348c;
const MTLS_STATUS = {
  Enforced: 'pass',
  Permissive: 'warn',
  Disabled: 'fail'
};
function Workloads({
  onOpenResource
}) {
  const d = window.OMG_DATA;
  return /*#__PURE__*/React.createElement(WCard, {
    padded: false,
    title: "Workloads",
    subtitle: `${d.workloads.length} mesh-enabled workloads · prod-eu-1`,
    actions: /*#__PURE__*/React.createElement(WButton, {
      size: "sm",
      variant: "secondary",
      leftIcon: /*#__PURE__*/React.createElement(Icon, {
        name: "download",
        size: 15
      })
    }, "Export")
  }, /*#__PURE__*/React.createElement(WTable, {
    rowKey: "name",
    onRowClick: r => onOpenResource({
      ...r,
      id: 'WL-' + r.name,
      title: `Workload posture: ${r.name}`,
      kind: 'Workload',
      resource: r.name,
      severity: MTLS_STATUS[r.mtls],
      sevLabel: r.mtls === 'Enforced' ? 'Low' : 'High',
      control: r.mtls === 'Enforced' ? 'mTLS' : 'mTLS',
      age: '2d'
    }),
    columns: [{
      key: 'name',
      header: 'Workload',
      mono: true
    }, {
      key: 'ns',
      header: 'Namespace',
      mono: true
    }, {
      key: 'owner',
      header: 'Owner',
      render: v => v === 'unowned' ? /*#__PURE__*/React.createElement(WStatus, {
        status: "warn"
      }, "Unowned") : /*#__PURE__*/React.createElement("span", {
        style: {
          display: 'inline-flex',
          alignItems: 'center',
          gap: 6
        }
      }, /*#__PURE__*/React.createElement(WAvatar, {
        name: v,
        size: "sm"
      }), /*#__PURE__*/React.createElement("span", {
        style: {
          fontFamily: 'var(--font-mono)',
          fontSize: 12
        }
      }, v))
    }, {
      key: 'mtls',
      header: 'mTLS',
      render: v => /*#__PURE__*/React.createElement(WStatus, {
        status: MTLS_STATUS[v]
      }, v)
    }, {
      key: 'authz',
      header: 'AuthZ',
      render: v => /*#__PURE__*/React.createElement(WStatus, {
        status: v === 'Present' ? 'pass' : 'fail'
      }, v)
    }, {
      key: 'gitops',
      header: 'GitOps',
      render: v => /*#__PURE__*/React.createElement(WStatus, {
        status: v === 'In sync' ? 'pass' : 'warn'
      }, v)
    }, {
      key: 'coverage',
      header: 'Coverage',
      width: 170,
      render: v => /*#__PURE__*/React.createElement(WProgress, {
        value: v
      })
    }],
    rows: d.workloads
  }));
}
window.Workloads = Workloads;
})(); } catch (e) { __ds_ns.__errors.push({ path: "ui_kits/posture-scanner/Workloads.jsx", error: String((e && e.message) || e) }); }

// ui_kits/posture-scanner/app.jsx
try { (() => {
// Posture Scanner app — routing + interactive state.
const {
  Toast: AppToast,
  Card: AppCard,
  DataTable: AppTable,
  StatusBadge: AppStatus,
  Badge: AppBadge,
  Button: AppButton,
  Avatar: AppAvatar
} = window.OpenMeshGuardDesignSystem_65348c;
const SCREEN_META = {
  overview: {
    title: 'Cluster posture overview',
    subtitle: 'Verified security posture across your Istio mesh'
  },
  findings: {
    title: null,
    subtitle: null
  },
  workloads: {
    title: null,
    subtitle: null
  },
  drift: {
    title: 'Configuration drift',
    subtitle: 'In-cluster policy vs the GitOps source of truth'
  },
  exceptions: {
    title: 'Exceptions',
    subtitle: 'Approved deviations and their lifecycle'
  },
  evidence: {
    title: 'Evidence & reports',
    subtitle: 'Generate audit-ready posture evidence'
  },
  settings: {
    title: 'Settings',
    subtitle: 'Scanning, controls, and integrations'
  }
};
function DriftScreen({
  onOpenResource
}) {
  const rows = window.OMG_DATA.findings.filter(f => f.control === 'Drift').concat([{
    id: 'OMG-1012',
    title: 'DestinationRule mTLS mode changed in-cluster',
    kind: 'DestinationRule',
    resource: 'db-mtls',
    ns: 'data',
    owner: 'team-data',
    severity: 'warn',
    sevLabel: 'Medium',
    control: 'Drift',
    age: '9d'
  }]);
  return /*#__PURE__*/React.createElement(AppCard, {
    padded: false,
    title: "Drifted resources",
    subtitle: `${rows.length} resources differ from Git`
  }, /*#__PURE__*/React.createElement(AppTable, {
    rowKey: "id",
    onRowClick: onOpenResource,
    columns: [{
      key: 'resource',
      header: 'Resource',
      mono: true
    }, {
      key: 'kind',
      header: 'Kind',
      render: v => /*#__PURE__*/React.createElement("code", {
        style: {
          fontFamily: 'var(--font-mono)',
          fontSize: 12
        }
      }, v)
    }, {
      key: 'ns',
      header: 'Namespace',
      mono: true
    }, {
      key: 'owner',
      header: 'Owner',
      mono: true
    }, {
      key: 'title',
      header: 'Change',
      render: v => /*#__PURE__*/React.createElement("span", {
        style: {
          whiteSpace: 'normal'
        }
      }, v)
    }, {
      key: 'status',
      header: 'GitOps',
      render: () => /*#__PURE__*/React.createElement(AppStatus, {
        status: "warn"
      }, "Drifted")
    }],
    rows: rows
  }));
}
function ExceptionsScreen() {
  const rows = [{
    name: 'permit-permissive-mtls',
    ns: 'checkout',
    owner: 'team-web',
    reason: 'Legacy client migration',
    expires: 'in 5 days',
    status: 'warn',
    statusLabel: 'Expiring'
  }, {
    name: 'allow-public-status',
    ns: 'edge',
    owner: 'team-platform',
    reason: 'Public health endpoint',
    expires: 'in 88 days',
    status: 'pass',
    statusLabel: 'Active'
  }, {
    name: 'skip-authz-batch',
    ns: 'ledger',
    owner: 'team-core',
    reason: 'Batch job identity',
    expires: '3 days ago',
    status: 'fail',
    statusLabel: 'Expired'
  }];
  return /*#__PURE__*/React.createElement(AppCard, {
    padded: false,
    title: "Exceptions",
    subtitle: `${rows.length} exceptions`,
    actions: /*#__PURE__*/React.createElement(AppButton, {
      size: "sm",
      variant: "primary",
      leftIcon: /*#__PURE__*/React.createElement(Icon, {
        name: "plus",
        size: 15
      })
    }, "Request exception")
  }, /*#__PURE__*/React.createElement(AppTable, {
    rowKey: "name",
    columns: [{
      key: 'name',
      header: 'Exception',
      mono: true
    }, {
      key: 'ns',
      header: 'Namespace',
      mono: true
    }, {
      key: 'owner',
      header: 'Owner',
      render: v => /*#__PURE__*/React.createElement("span", {
        style: {
          display: 'inline-flex',
          alignItems: 'center',
          gap: 6
        }
      }, /*#__PURE__*/React.createElement(AppAvatar, {
        name: v,
        size: "sm"
      }), /*#__PURE__*/React.createElement("span", {
        style: {
          fontFamily: 'var(--font-mono)',
          fontSize: 12
        }
      }, v))
    }, {
      key: 'reason',
      header: 'Justification',
      render: v => /*#__PURE__*/React.createElement("span", {
        style: {
          whiteSpace: 'normal'
        }
      }, v)
    }, {
      key: 'expires',
      header: 'Expires',
      mono: true
    }, {
      key: 'status',
      header: 'Status',
      render: (v, r) => /*#__PURE__*/React.createElement(AppStatus, {
        status: r.status
      }, r.statusLabel)
    }],
    rows: rows
  }));
}
function SettingsScreen() {
  return /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'grid',
      gridTemplateColumns: '1fr 1fr',
      gap: 20,
      alignItems: 'start'
    }
  }, /*#__PURE__*/React.createElement(AppCard, {
    title: "Scanning"
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      flexDirection: 'column',
      gap: 14
    }
  }, /*#__PURE__*/React.createElement(SettingRow, {
    label: "Continuous scanning",
    desc: "Re-scan every 4 hours",
    defaultOn: true
  }), /*#__PURE__*/React.createElement(SettingRow, {
    label: "Scan on GitOps change",
    desc: "Trigger a scan when Git changes",
    defaultOn: true
  }), /*#__PURE__*/React.createElement(SettingRow, {
    label: "Ambient mesh readiness",
    desc: "Flag workloads blocking migration"
  }))), /*#__PURE__*/React.createElement(AppCard, {
    title: "Integrations"
  }, /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      flexDirection: 'column',
      gap: 12
    }
  }, [['GitOps repository', 'git.corp/mesh', 'pass', 'Connected'], ['Identity provider', 'okta.corp', 'pass', 'Connected'], ['Ticketing', 'Not configured', 'none', 'Off']].map(([a, b, s, l]) => /*#__PURE__*/React.createElement("div", {
    key: a,
    style: {
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'space-between',
      padding: '10px 0',
      borderBottom: '1px solid var(--border-subtle)'
    }
  }, /*#__PURE__*/React.createElement("div", null, /*#__PURE__*/React.createElement("div", {
    style: {
      fontSize: 14,
      color: 'var(--text-strong)',
      fontWeight: 500
    }
  }, a), /*#__PURE__*/React.createElement("code", {
    style: {
      fontFamily: 'var(--font-mono)',
      fontSize: 12,
      color: 'var(--text-muted)'
    }
  }, b)), /*#__PURE__*/React.createElement(AppStatus, {
    status: s
  }, l))))));
}
function SettingRow({
  label,
  desc,
  defaultOn
}) {
  const {
    Switch: Sw
  } = window.OpenMeshGuardDesignSystem_65348c;
  const [on, setOn] = React.useState(!!defaultOn);
  return /*#__PURE__*/React.createElement("div", {
    style: {
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'space-between',
      gap: 16,
      paddingBottom: 14,
      borderBottom: '1px solid var(--border-subtle)'
    }
  }, /*#__PURE__*/React.createElement("div", null, /*#__PURE__*/React.createElement("div", {
    style: {
      fontSize: 14,
      color: 'var(--text-strong)',
      fontWeight: 500
    }
  }, label), /*#__PURE__*/React.createElement("div", {
    style: {
      fontSize: 12,
      color: 'var(--text-muted)'
    }
  }, desc)), /*#__PURE__*/React.createElement(Sw, {
    checked: on,
    onChange: setOn
  }));
}
function App() {
  const [active, setActive] = React.useState('overview');
  const [finding, setFinding] = React.useState(null);
  const [toast, setToast] = React.useState(false);
  const [cluster, setCluster] = React.useState('prod-eu-1');
  const openResource = f => {
    setFinding(f);
    setActive('resource');
  };
  const nav = id => {
    setFinding(null);
    setActive(id);
  };
  const runScan = () => {
    setToast(true);
    setTimeout(() => setToast(false), 3200);
  };
  let screen, meta;
  if (active === 'resource') {
    screen = /*#__PURE__*/React.createElement(ResourceDetail, {
      finding: finding,
      onBack: () => setActive('findings')
    });
    meta = {
      title: null
    };
  } else {
    meta = SCREEN_META[active] || {};
    screen = {
      overview: /*#__PURE__*/React.createElement(Overview, {
        onOpenFindings: () => nav('findings'),
        onOpenResource: openResource
      }),
      findings: /*#__PURE__*/React.createElement(Findings, {
        onOpenResource: openResource
      }),
      workloads: /*#__PURE__*/React.createElement(Workloads, {
        onOpenResource: openResource
      }),
      drift: /*#__PURE__*/React.createElement(DriftScreen, {
        onOpenResource: openResource
      }),
      exceptions: /*#__PURE__*/React.createElement(ExceptionsScreen, null),
      evidence: /*#__PURE__*/React.createElement(Evidence, null),
      settings: /*#__PURE__*/React.createElement(SettingsScreen, null)
    }[active] || /*#__PURE__*/React.createElement(Overview, {
      onOpenFindings: () => nav('findings'),
      onOpenResource: openResource
    });
  }
  return /*#__PURE__*/React.createElement(React.Fragment, null, /*#__PURE__*/React.createElement(AppShell, {
    active: active === 'resource' ? 'findings' : active,
    onNav: nav,
    cluster: cluster,
    onCluster: setCluster,
    onScan: runScan,
    title: meta.title,
    subtitle: meta.subtitle
  }, screen), toast && /*#__PURE__*/React.createElement("div", {
    style: {
      position: 'fixed',
      bottom: 24,
      right: 24,
      zIndex: 100
    }
  }, /*#__PURE__*/React.createElement(AppToast, {
    status: "pass",
    title: "Scan complete",
    message: `${cluster} scanned. 3 new findings.`,
    onDismiss: () => setToast(false)
  })));
}
ReactDOM.createRoot(document.getElementById('root')).render(/*#__PURE__*/React.createElement(App, null));
})(); } catch (e) { __ds_ns.__errors.push({ path: "ui_kits/posture-scanner/app.jsx", error: String((e && e.message) || e) }); }

// ui_kits/posture-scanner/data.jsx
try { (() => {
// Shared mock data for the OpenMeshGuard Posture Scanner UI kit.

window.OMG_DATA = {
  clusters: ['prod-eu-1', 'prod-us-1', 'staging-eu-1'],
  metrics: {
    mtlsCoverage: 82,
    openFindings: 37,
    critical: 12,
    ownedPct: 94,
    exceptions: 8,
    drifted: 5
  },
  severityBreakdown: [{
    label: 'Critical',
    count: 12,
    status: 'fail'
  }, {
    label: 'High',
    count: 9,
    status: 'fail'
  }, {
    label: 'Medium',
    count: 11,
    status: 'warn'
  }, {
    label: 'Low',
    count: 5,
    status: 'info'
  }],
  controls: [{
    name: 'Strict mTLS in production',
    coverage: 82,
    status: 'warn'
  }, {
    name: 'AuthorizationPolicy present',
    coverage: 71,
    status: 'warn'
  }, {
    name: 'No public exposure of internal services',
    coverage: 96,
    status: 'pass'
  }, {
    name: 'Ownership metadata complete',
    coverage: 94,
    status: 'pass'
  }, {
    name: 'GitOps in sync (no drift)',
    coverage: 88,
    status: 'warn'
  }],
  findings: [{
    id: 'OMG-1042',
    title: 'Ingress Gateway exposes an internal service to the public',
    kind: 'Gateway',
    resource: 'checkout-gateway',
    ns: 'checkout',
    owner: 'team-web',
    severity: 'fail',
    sevLabel: 'Critical',
    control: 'Exposure',
    age: '2d'
  }, {
    id: 'OMG-1039',
    title: 'Namespace not enforcing strict mTLS',
    kind: 'PeerAuthentication',
    resource: 'ledger',
    ns: 'ledger',
    owner: 'team-core',
    severity: 'fail',
    sevLabel: 'Critical',
    control: 'mTLS',
    age: '2d'
  }, {
    id: 'OMG-1031',
    title: 'Mesh-enabled app has no AuthorizationPolicy',
    kind: 'Workload',
    resource: 'ledger-svc',
    ns: 'ledger',
    owner: 'team-core',
    severity: 'fail',
    sevLabel: 'High',
    control: 'Authorization',
    age: '4d'
  }, {
    id: 'OMG-1028',
    title: 'VirtualService routes to a workload outside the mesh',
    kind: 'VirtualService',
    resource: 'legacy-router',
    ns: 'edge',
    owner: 'unowned',
    severity: 'warn',
    sevLabel: 'High',
    control: 'Exposure',
    age: '5d'
  }, {
    id: 'OMG-1024',
    title: 'Istio resource missing owner and repo metadata',
    kind: 'DestinationRule',
    resource: 'db-mtls',
    ns: 'data',
    owner: 'unowned',
    severity: 'warn',
    sevLabel: 'Medium',
    control: 'Ownership',
    age: '6d'
  }, {
    id: 'OMG-1019',
    title: 'Policy drifted from GitOps source of truth',
    kind: 'AuthorizationPolicy',
    resource: 'payments-allow',
    ns: 'payments',
    owner: 'team-payments',
    severity: 'warn',
    sevLabel: 'Medium',
    control: 'Drift',
    age: '8d'
  }, {
    id: 'OMG-1004',
    title: 'Exception expires in 5 days',
    kind: 'Exception',
    resource: 'permit-permissive-mtls',
    ns: 'checkout',
    owner: 'team-web',
    severity: 'info',
    sevLabel: 'Low',
    control: 'Exception',
    age: '12d'
  }],
  workloads: [{
    name: 'payments-api',
    ns: 'payments',
    owner: 'team-payments',
    mtls: 'Enforced',
    authz: 'Present',
    coverage: 98,
    gitops: 'In sync'
  }, {
    name: 'checkout-web',
    ns: 'checkout',
    owner: 'team-web',
    mtls: 'Permissive',
    authz: 'Present',
    coverage: 64,
    gitops: 'In sync'
  }, {
    name: 'ledger-svc',
    ns: 'ledger',
    owner: 'team-core',
    mtls: 'Disabled',
    authz: 'Missing',
    coverage: 22,
    gitops: 'Drifted'
  }, {
    name: 'search-api',
    ns: 'search',
    owner: 'team-discovery',
    mtls: 'Enforced',
    authz: 'Present',
    coverage: 91,
    gitops: 'In sync'
  }, {
    name: 'legacy-router',
    ns: 'edge',
    owner: 'unowned',
    mtls: 'Permissive',
    authz: 'Missing',
    coverage: 40,
    gitops: 'Drifted'
  }]
};
})(); } catch (e) { __ds_ns.__errors.push({ path: "ui_kits/posture-scanner/data.jsx", error: String((e && e.message) || e) }); }

__ds_ns.Button = __ds_scope.Button;

__ds_ns.IconButton = __ds_scope.IconButton;

__ds_ns.Avatar = __ds_scope.Avatar;

__ds_ns.Card = __ds_scope.Card;

__ds_ns.DataTable = __ds_scope.DataTable;

__ds_ns.Tabs = __ds_scope.Tabs;

__ds_ns.Toast = __ds_scope.Toast;

__ds_ns.Tooltip = __ds_scope.Tooltip;

__ds_ns.Checkbox = __ds_scope.Checkbox;

__ds_ns.Input = __ds_scope.Input;

__ds_ns.Select = __ds_scope.Select;

__ds_ns.Switch = __ds_scope.Switch;

__ds_ns.Badge = __ds_scope.Badge;

__ds_ns.MetricStat = __ds_scope.MetricStat;

__ds_ns.ProgressBar = __ds_scope.ProgressBar;

__ds_ns.StatusBadge = __ds_scope.StatusBadge;

__ds_ns.Tag = __ds_scope.Tag;

})();

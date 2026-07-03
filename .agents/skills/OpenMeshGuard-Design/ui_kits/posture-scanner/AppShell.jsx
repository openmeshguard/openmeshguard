// App shell: fixed sidebar + topbar, scrollable content.
const { Button, IconButton, StatusBadge, Select, Input } = window.OpenMeshGuardDesignSystem_65348c;

const NAV = [
  { id: 'overview', label: 'Overview', icon: 'dashboard' },
  { id: 'findings', label: 'Findings', icon: 'triangle-alert', count: 37 },
  { id: 'workloads', label: 'Workloads', icon: 'network' },
  { id: 'drift', label: 'Drift', icon: 'git-compare', count: 5 },
  { id: 'exceptions', label: 'Exceptions', icon: 'clock' },
  { id: 'evidence', label: 'Evidence', icon: 'file-check' },
];

function NavItem({ item, active, onClick }) {
  const [hover, setHover] = React.useState(false);
  return (
    <button
      onClick={onClick}
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
      style={{
        display: 'flex', alignItems: 'center', gap: 10, width: '100%',
        padding: '8px 10px', border: 'none', cursor: 'pointer', textAlign: 'left',
        borderRadius: 'var(--radius-md)', fontFamily: 'var(--font-sans)', fontSize: 14,
        fontWeight: active ? 600 : 500,
        color: active ? 'var(--brand-700)' : hover ? 'var(--text-strong)' : 'var(--text-body)',
        background: active ? 'var(--brand-50)' : hover ? 'var(--surface-hover)' : 'transparent',
        transition: 'background var(--dur-fast), color var(--dur-fast)',
      }}
    >
      <Icon name={item.icon} size={17} style={{ color: active ? 'var(--brand-600)' : 'var(--text-muted)' }} />
      <span style={{ flex: 1 }}>{item.label}</span>
      {item.count != null && (
        <span style={{ fontFamily: 'var(--font-mono)', fontSize: 11, fontWeight: 600, color: active ? 'var(--brand-700)' : 'var(--text-muted)' }}>{item.count}</span>
      )}
    </button>
  );
}

function AppShell({ active, onNav, cluster, onCluster, onScan, children, title, subtitle, actions }) {
  return (
    <div style={{ display: 'flex', height: '100%', background: 'var(--surface-page)', color: 'var(--text-body)', fontFamily: 'var(--font-sans)' }}>
      {/* Sidebar */}
      <aside style={{ width: 'var(--sidebar-width)', flex: 'none', background: 'var(--surface-card)', borderRight: '1px solid var(--border-subtle)', display: 'flex', flexDirection: 'column' }}>
        <div style={{ height: 'var(--topbar-height)', display: 'flex', alignItems: 'center', padding: '0 16px', borderBottom: '1px solid var(--border-subtle)' }}>
          <img src="../../assets/logo-wordmark.svg" height="26" alt="OpenMeshGuard" />
        </div>
        <nav style={{ padding: 12, display: 'flex', flexDirection: 'column', gap: 2, flex: 1 }}>
          <div className="omg-eyebrow" style={{ padding: '8px 10px 4px' }}>Posture</div>
          {NAV.map((it) => <NavItem key={it.id} item={it} active={active === it.id} onClick={() => onNav(it.id)} />)}
        </nav>
        <div style={{ padding: 12, borderTop: '1px solid var(--border-subtle)' }}>
          <NavItem item={{ id: 'settings', label: 'Settings', icon: 'settings' }} active={active === 'settings'} onClick={() => onNav('settings')} />
        </div>
      </aside>

      {/* Main */}
      <div style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
        {/* Topbar */}
        <header style={{ height: 'var(--topbar-height)', flex: 'none', display: 'flex', alignItems: 'center', gap: 12, padding: '0 24px', background: 'var(--surface-card)', borderBottom: '1px solid var(--border-subtle)' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, color: 'var(--text-muted)', fontSize: 13 }}>
            <Icon name="box" size={15} />
            <span>Cluster</span>
          </div>
          <div style={{ width: 150 }}>
            <Select value={cluster} onChange={(e) => onCluster(e.target.value)} size="sm"
              options={window.OMG_DATA.clusters.map((c) => ({ value: c, label: c }))} />
          </div>
          <div style={{ flex: 1, maxWidth: 320, marginLeft: 8 }}>
            <Input size="sm" placeholder="Search resources, owners, namespaces" leadingIcon={<Icon name="search" size={15} />} />
          </div>
          <div style={{ flex: 1 }} />
          <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6, fontSize: 12, color: 'var(--text-muted)' }}>
            <Icon name="refresh" size={14} /> Last scan 4h ago
          </span>
          <IconButton title="Notifications" variant="ghost"><Icon name="bell" size={18} /></IconButton>
          <Button size="sm" variant="primary" leftIcon={<Icon name="refresh" size={15} />} onClick={onScan}>Run scan</Button>
        </header>

        {/* Page header + content */}
        <main style={{ flex: 1, overflowY: 'auto', padding: '24px' }}>
          <div style={{ maxWidth: 'var(--content-max)', margin: '0 auto' }}>
            {(title || actions) && (
              <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 16, marginBottom: 20 }}>
                <div>
                  {title && <h1 style={{ fontSize: 24, fontWeight: 600, color: 'var(--text-strong)' }}>{title}</h1>}
                  {subtitle && <p style={{ fontSize: 14, color: 'var(--text-muted)', marginTop: 4 }}>{subtitle}</p>}
                </div>
                {actions && <div style={{ display: 'flex', gap: 8, flex: 'none' }}>{actions}</div>}
              </div>
            )}
            {children}
          </div>
        </main>
      </div>
    </div>
  );
}

window.AppShell = AppShell;

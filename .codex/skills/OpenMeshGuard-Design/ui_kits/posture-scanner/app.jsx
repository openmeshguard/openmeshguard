// Posture Scanner app — routing + interactive state.
const { Toast: AppToast, Card: AppCard, DataTable: AppTable, StatusBadge: AppStatus, Badge: AppBadge, Button: AppButton, Avatar: AppAvatar } = window.OpenMeshGuardDesignSystem_65348c;

const SCREEN_META = {
  overview: { title: 'Cluster posture overview', subtitle: 'Verified security posture across your Istio mesh' },
  findings: { title: null, subtitle: null },
  workloads: { title: null, subtitle: null },
  drift: { title: 'Configuration drift', subtitle: 'In-cluster policy vs the GitOps source of truth' },
  exceptions: { title: 'Exceptions', subtitle: 'Approved deviations and their lifecycle' },
  evidence: { title: 'Evidence & reports', subtitle: 'Generate audit-ready posture evidence' },
  settings: { title: 'Settings', subtitle: 'Scanning, controls, and integrations' },
};

function DriftScreen({ onOpenResource }) {
  const rows = window.OMG_DATA.findings.filter((f) => f.control === 'Drift').concat([
    { id: 'OMG-1012', title: 'DestinationRule mTLS mode changed in-cluster', kind: 'DestinationRule', resource: 'db-mtls', ns: 'data', owner: 'team-data', severity: 'warn', sevLabel: 'Medium', control: 'Drift', age: '9d' },
  ]);
  return (
    <AppCard padded={false} title="Drifted resources" subtitle={`${rows.length} resources differ from Git`}>
      <AppTable rowKey="id" onRowClick={onOpenResource}
        columns={[
          { key: 'resource', header: 'Resource', mono: true },
          { key: 'kind', header: 'Kind', render: (v) => <code style={{ fontFamily: 'var(--font-mono)', fontSize: 12 }}>{v}</code> },
          { key: 'ns', header: 'Namespace', mono: true },
          { key: 'owner', header: 'Owner', mono: true },
          { key: 'title', header: 'Change', render: (v) => <span style={{ whiteSpace: 'normal' }}>{v}</span> },
          { key: 'status', header: 'GitOps', render: () => <AppStatus status="warn">Drifted</AppStatus> },
        ]}
        rows={rows} />
    </AppCard>
  );
}

function ExceptionsScreen() {
  const rows = [
    { name: 'permit-permissive-mtls', ns: 'checkout', owner: 'team-web', reason: 'Legacy client migration', expires: 'in 5 days', status: 'warn', statusLabel: 'Expiring' },
    { name: 'allow-public-status', ns: 'edge', owner: 'team-platform', reason: 'Public health endpoint', expires: 'in 88 days', status: 'pass', statusLabel: 'Active' },
    { name: 'skip-authz-batch', ns: 'ledger', owner: 'team-core', reason: 'Batch job identity', expires: '3 days ago', status: 'fail', statusLabel: 'Expired' },
  ];
  return (
    <AppCard padded={false} title="Exceptions" subtitle={`${rows.length} exceptions`}
      actions={<AppButton size="sm" variant="primary" leftIcon={<Icon name="plus" size={15} />}>Request exception</AppButton>}>
      <AppTable rowKey="name"
        columns={[
          { key: 'name', header: 'Exception', mono: true },
          { key: 'ns', header: 'Namespace', mono: true },
          { key: 'owner', header: 'Owner', render: (v) => <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}><AppAvatar name={v} size="sm" /><span style={{ fontFamily: 'var(--font-mono)', fontSize: 12 }}>{v}</span></span> },
          { key: 'reason', header: 'Justification', render: (v) => <span style={{ whiteSpace: 'normal' }}>{v}</span> },
          { key: 'expires', header: 'Expires', mono: true },
          { key: 'status', header: 'Status', render: (v, r) => <AppStatus status={r.status}>{r.statusLabel}</AppStatus> },
        ]}
        rows={rows} />
    </AppCard>
  );
}

function SettingsScreen() {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 20, alignItems: 'start' }}>
      <AppCard title="Scanning">
        <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
          <SettingRow label="Continuous scanning" desc="Re-scan every 4 hours" defaultOn />
          <SettingRow label="Scan on GitOps change" desc="Trigger a scan when Git changes" defaultOn />
          <SettingRow label="Ambient mesh readiness" desc="Flag workloads blocking migration" />
        </div>
      </AppCard>
      <AppCard title="Integrations">
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          {[['GitOps repository', 'git.corp/mesh', 'pass', 'Connected'], ['Identity provider', 'okta.corp', 'pass', 'Connected'], ['Ticketing', 'Not configured', 'none', 'Off']].map(([a, b, s, l]) => (
            <div key={a} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '10px 0', borderBottom: '1px solid var(--border-subtle)' }}>
              <div><div style={{ fontSize: 14, color: 'var(--text-strong)', fontWeight: 500 }}>{a}</div><code style={{ fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--text-muted)' }}>{b}</code></div>
              <AppStatus status={s}>{l}</AppStatus>
            </div>
          ))}
        </div>
      </AppCard>
    </div>
  );
}

function SettingRow({ label, desc, defaultOn }) {
  const { Switch: Sw } = window.OpenMeshGuardDesignSystem_65348c;
  const [on, setOn] = React.useState(!!defaultOn);
  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 16, paddingBottom: 14, borderBottom: '1px solid var(--border-subtle)' }}>
      <div><div style={{ fontSize: 14, color: 'var(--text-strong)', fontWeight: 500 }}>{label}</div><div style={{ fontSize: 12, color: 'var(--text-muted)' }}>{desc}</div></div>
      <Sw checked={on} onChange={setOn} />
    </div>
  );
}

function App() {
  const [active, setActive] = React.useState('overview');
  const [finding, setFinding] = React.useState(null);
  const [toast, setToast] = React.useState(false);
  const [cluster, setCluster] = React.useState('prod-eu-1');

  const openResource = (f) => { setFinding(f); setActive('resource'); };
  const nav = (id) => { setFinding(null); setActive(id); };
  const runScan = () => { setToast(true); setTimeout(() => setToast(false), 3200); };

  let screen, meta;
  if (active === 'resource') {
    screen = <ResourceDetail finding={finding} onBack={() => setActive('findings')} />;
    meta = { title: null };
  } else {
    meta = SCREEN_META[active] || {};
    screen = {
      overview: <Overview onOpenFindings={() => nav('findings')} onOpenResource={openResource} />,
      findings: <Findings onOpenResource={openResource} />,
      workloads: <Workloads onOpenResource={openResource} />,
      drift: <DriftScreen onOpenResource={openResource} />,
      exceptions: <ExceptionsScreen />,
      evidence: <Evidence />,
      settings: <SettingsScreen />,
    }[active] || <Overview onOpenFindings={() => nav('findings')} onOpenResource={openResource} />;
  }

  return (
    <>
      <AppShell active={active === 'resource' ? 'findings' : active} onNav={nav}
        cluster={cluster} onCluster={setCluster} onScan={runScan}
        title={meta.title} subtitle={meta.subtitle}>
        {screen}
      </AppShell>
      {toast && (
        <div style={{ position: 'fixed', bottom: 24, right: 24, zIndex: 100 }}>
          <AppToast status="pass" title="Scan complete" message={`${cluster} scanned. 3 new findings.`} onDismiss={() => setToast(false)} />
        </div>
      )}
    </>
  );
}

ReactDOM.createRoot(document.getElementById('root')).render(<App />);

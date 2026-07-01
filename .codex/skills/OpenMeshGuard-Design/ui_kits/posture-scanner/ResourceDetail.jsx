// Resource / finding detail drill-in.
const { Card: RCard, StatusBadge: RStatus, Tag: RTag, Button: RButton, Avatar: RAvatar, ProgressBar: RProgress } = window.OpenMeshGuardDesignSystem_65348c;

function Row({ label, children }) {
  return (
    <div style={{ display: 'flex', gap: 16, padding: '10px 0', borderBottom: '1px solid var(--border-subtle)' }}>
      <span style={{ width: 150, flex: 'none', fontSize: 13, color: 'var(--text-muted)' }}>{label}</span>
      <div style={{ flex: 1, fontSize: 13, color: 'var(--text-body)' }}>{children}</div>
    </div>
  );
}

function ResourceDetail({ finding, onBack }) {
  const f = finding || window.OMG_DATA.findings[0];
  const yaml = `apiVersion: security.istio.io/v1
kind: PeerAuthentication
metadata:
  name: default
  namespace: ${f.ns}
spec:
  mtls:
    mode: PERMISSIVE   # expected: STRICT`;

  const checks = [
    { name: 'Strict mTLS enforced', status: f.control === 'mTLS' ? 'fail' : 'pass' },
    { name: 'AuthorizationPolicy present', status: f.control === 'Authorization' ? 'fail' : 'pass' },
    { name: 'Not publicly exposed', status: f.control === 'Exposure' ? 'fail' : 'pass' },
    { name: 'Ownership metadata complete', status: f.owner === 'unowned' ? 'fail' : 'pass' },
    { name: 'In sync with GitOps', status: f.control === 'Drift' ? 'fail' : 'pass' },
  ];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
      <button onClick={onBack} style={{ display: 'inline-flex', alignItems: 'center', gap: 6, background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', fontSize: 13, padding: 0, width: 'fit-content', fontFamily: 'var(--font-sans)' }}>
        <Icon name="arrow-left" size={15} /> Back to findings
      </button>

      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 16 }}>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <RStatus status={f.severity} solid={f.severity === 'fail'}>{f.sevLabel}</RStatus>
            <code style={{ fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--text-muted)' }}>{f.id}</code>
          </div>
          <h1 style={{ fontSize: 22, fontWeight: 600, color: 'var(--text-strong)', maxWidth: 720 }}>{f.title}</h1>
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            <RTag label="kind" value={f.kind} />
            <RTag label="resource" value={`${f.ns}/${f.resource}`} />
            <RTag label="control" value={f.control} />
          </div>
        </div>
        <div style={{ display: 'flex', gap: 8, flex: 'none' }}>
          <RButton variant="secondary" leftIcon={<Icon name="user" size={15} />}>Assign owner</RButton>
          <RButton variant="secondary" leftIcon={<Icon name="clock" size={15} />}>Request exception</RButton>
          <RButton variant="primary" leftIcon={<Icon name="download" size={15} />}>Export evidence</RButton>
        </div>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 20, alignItems: 'start' }}>
        {/* Control checklist */}
        <RCard title="Control posture" subtitle="Evaluated against enterprise controls">
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            {checks.map((c) => (
              <div key={c.name} style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                <Icon name={c.status === 'pass' ? 'shield-check' : 'shield-alert'} size={18} style={{ color: `var(--status-${c.status}-solid)` }} />
                <span style={{ flex: 1, fontSize: 14, color: 'var(--text-strong)' }}>{c.name}</span>
                <RStatus status={c.status}>{c.status === 'pass' ? 'Pass' : 'Fail'}</RStatus>
              </div>
            ))}
          </div>
        </RCard>

        {/* Metadata & ownership */}
        <RCard title="Ownership & metadata">
          <Row label="Owner">{f.owner === 'unowned' ? <RStatus status="warn">Unowned</RStatus> : <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}><RAvatar name={f.owner} size="sm" /><code style={{ fontFamily: 'var(--font-mono)', fontSize: 12 }}>{f.owner}</code></span>}</Row>
          <Row label="Namespace"><code style={{ fontFamily: 'var(--font-mono)' }}>{f.ns}</code></Row>
          <Row label="Environment"><code style={{ fontFamily: 'var(--font-mono)' }}>production</code></Row>
          <Row label="Repository">{f.owner === 'unowned' ? <RStatus status="warn">Missing</RStatus> : <a href="#">git.corp/mesh/{f.resource}</a>}</Row>
          <Row label="First detected">{f.age} ago</Row>
        </RCard>
      </div>

      {/* Evidence */}
      <RCard title="Evidence" subtitle="Observed in-cluster configuration"
        actions={<RButton size="sm" variant="ghost" leftIcon={<Icon name="external-link" size={14} />}>Open in cluster</RButton>}>
        <pre style={{ margin: 0, fontFamily: 'var(--font-mono)', fontSize: 13, lineHeight: 1.6, color: 'var(--slate-100)', background: 'var(--slate-900)', padding: 16, borderRadius: 'var(--radius-md)', overflowX: 'auto' }}>{yaml}</pre>
      </RCard>
    </div>
  );
}

window.ResourceDetail = ResourceDetail;

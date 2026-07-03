// Workloads inventory screen.
const { Card: WCard, DataTable: WTable, StatusBadge: WStatus, Avatar: WAvatar, ProgressBar: WProgress, Button: WButton } = window.OpenMeshGuardDesignSystem_65348c;

const MTLS_STATUS = { Enforced: 'pass', Permissive: 'warn', Disabled: 'fail' };

function Workloads({ onOpenResource }) {
  const d = window.OMG_DATA;
  return (
    <WCard padded={false} title="Workloads" subtitle={`${d.workloads.length} mesh-enabled workloads · prod-eu-1`}
      actions={<WButton size="sm" variant="secondary" leftIcon={<Icon name="download" size={15} />}>Export</WButton>}>
      <WTable
        rowKey="name"
        onRowClick={(r) => onOpenResource({ ...r, id: 'WL-' + r.name, title: `Workload posture: ${r.name}`, kind: 'Workload', resource: r.name, severity: MTLS_STATUS[r.mtls], sevLabel: r.mtls === 'Enforced' ? 'Low' : 'High', control: r.mtls === 'Enforced' ? 'mTLS' : 'mTLS', age: '2d' })}
        columns={[
          { key: 'name', header: 'Workload', mono: true },
          { key: 'ns', header: 'Namespace', mono: true },
          { key: 'owner', header: 'Owner', render: (v) => v === 'unowned'
            ? <WStatus status="warn">Unowned</WStatus>
            : <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}><WAvatar name={v} size="sm" /><span style={{ fontFamily: 'var(--font-mono)', fontSize: 12 }}>{v}</span></span> },
          { key: 'mtls', header: 'mTLS', render: (v) => <WStatus status={MTLS_STATUS[v]}>{v}</WStatus> },
          { key: 'authz', header: 'AuthZ', render: (v) => <WStatus status={v === 'Present' ? 'pass' : 'fail'}>{v}</WStatus> },
          { key: 'gitops', header: 'GitOps', render: (v) => <WStatus status={v === 'In sync' ? 'pass' : 'warn'}>{v}</WStatus> },
          { key: 'coverage', header: 'Coverage', width: 170, render: (v) => <WProgress value={v} /> },
        ]}
        rows={d.workloads}
      />
    </WCard>
  );
}

window.Workloads = Workloads;

Interactive button for primary and secondary actions. Use for any command the user triggers.

```jsx
<Button variant="primary" onClick={runScan}>Run scan</Button>
<Button variant="secondary" leftIcon={<DownloadIcon/>}>Export evidence (PDF)</Button>
<Button variant="ghost" size="sm">Cancel</Button>
<Button variant="danger">Revoke exception</Button>
```

Variants: `primary` (guard blue, one per view), `secondary` (bordered), `ghost` (borderless), `danger` (red, destructive). Sizes: `sm` / `md` / `lg`. Labels are always sentence case. For icon-only use `IconButton`.

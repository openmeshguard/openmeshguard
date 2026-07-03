Status pill for control outcomes and lifecycle. Color + text always together (never color alone).

```jsx
<StatusBadge status="pass">Enforced</StatusBadge>
<StatusBadge status="warn">Expiring</StatusBadge>
<StatusBadge status="fail">Exposed</StatusBadge>
<StatusBadge status="none">Not scanned</StatusBadge>
<StatusBadge status="pass" solid>Pass</StatusBadge>
```

Statuses: `pass` `warn` `fail` `info` `none`. `solid` for high-emphasis rows; `dot={false}` to hide the leading dot.

The packet-life track: dots colored by event type (plan grey, edit/comment cyan, test amber, mutation/flag purple, held red), `held` events render big with a glow.

```jsx
<Timeline
  events={[
    { t: '09:02', type: 'plan', label: 'plan approved', rev: 'rev0.5' },
    { t: '09:21', type: 'mutation', label: '1 mutant survived', rev: 'rev1' },
    { t: 'now', type: 'held', label: 'held for review' },
  ]}
  selected={sel} onSelect={setSel}
/>
```

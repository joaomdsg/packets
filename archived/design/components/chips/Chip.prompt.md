Pill mono microtype chip: state chips are color-mix tinted + uppercase ("HELD · STRICT", "DELIVERED"); neutral chips are flat raised (addrs, revs).

```jsx
<Chip state="risk" bordered>HELD · STRICT</Chip>
<Chip state="delivered">DELIVERED</Chip>
<Chip>acme/edge-gateway</Chip>
<Chip size="sm" active onClick={pick}>acme/billing</Chip>
<Chip state="agent" dot>agent</Chip>
```

States: signal/you, verified, held, risk, agent, delivered, neutral. Sizes sm/md/lg (15/17/20px). Never invent a new state color.

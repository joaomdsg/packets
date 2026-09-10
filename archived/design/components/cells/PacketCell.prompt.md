The atomic packet-state square: state is color, live things pulse, `composing` renders the locked ghost outline (auto-fallback to solid below 14px).

```jsx
<PacketCell state="verified" size={8} />
<PacketCell state="risk" size={11} live />   // a held packet, pulsing
<PacketCell state="composing" size={22} />   // the ghost
```

States: composing (outline), inflight/you (cyan), verified (green), held (amber), risk (red), delivered (dark cyan), agent (purple). `round` renders a dot for live-status use.

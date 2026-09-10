The locked 2×2 packets mark — TL/BL signal, TR ghost (composing), BR delivered — with optional wordmark or stacked product lockup. Never draw the mark by hand; mount this.

```jsx
<PacketMark cell={8} />                          // mark only (app chrome)
<PacketMark cell={8} sub="inspector" />          // stacked packets/INSPECTOR lockup
<PacketMark cell={22} wordmark wordmarkSize={44} /> // brand lockup
<PacketMark cell={8} held />                     // live variant: a packet is held
```

Small-size ghost fallback is automatic (inherited from PacketCell). `held` is the "mark is the console" live variant.

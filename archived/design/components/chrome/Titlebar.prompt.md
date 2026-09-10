The locked titlebar: mark + stacked `packets/APP` lockup, hairline, an 11px status packet (hover → tooltip), then name + rev chip with addr below. Right side takes the annotations legend.

```jsx
<Titlebar
  app="inspector" name="rate-limiter" rev="rev2" addr="acme/edge-gateway"
  status={{ state: 'risk', label: 'HELD · STRICT', detail: 'held 34m · handshake below lane floor' }}
  right={<>
    <span className="mono" style={{fontSize:10,fontWeight:600,color:'var(--signal)'}}>✎ you · 2</span>
    <span className="mono" style={{fontSize:10,fontWeight:600,color:'var(--agent)'}}>⚑ agents · 2</span>
  </>}
/>
```

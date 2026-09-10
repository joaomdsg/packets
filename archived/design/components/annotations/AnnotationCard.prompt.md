Annotation card: author chip + severity chip + location, sans-prose body, optional scope/actions footers. Author color owns the 3px left border.

```jsx
<AnnotationCard author="agent" sev="self-flag" where="L25" scope="line · limiter.go"
  actions="reply · resolve · promote to handshake term →">
  Unsure about this boundary — inclusive (&gt;=) or exclusive (&gt;)? The handshake only gives examples.
</AnnotationCard>
```

Severities: blocking, self-flag, question, nit.

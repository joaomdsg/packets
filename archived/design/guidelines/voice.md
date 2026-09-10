# Packets — Voice & Copy

The system reports to an operator. It is calm, precise, and a little dry. It never performs enthusiasm.

## Rules

1. **Mono is the machine's voice.** Any string the *system* says — states, counts, labels, meta — is Plex Mono. Sans is reserved for human prose (annotation bodies, marketing paragraphs).
2. **Lowercase by default.** Panel labels and kickers are lowercase-or-tracked-caps mono ("needs you · 4", "IN FLIGHT · 8"). Meta phrases start lowercase: "held 34m", "the whole job", "last 24h". Marketing headlines are sentence case.
3. **Networking vocabulary, always.** forward, hold, inspect, deliver, ACK, drop, addr, lane, packet, stream, line rate, handshake, clause. Banned: PR, merge queue, approve, review-as-noun, LGTM.
4. **No exclamation marks. No emoji.** Ever.
5. **Second person, direct.** "nothing needs you" · "held for you" · "you never rubber-stamp". The user is *you*; agents are named ("agent-S") or just "agents".
6. **Counts plainly, `·` separated.** "4 files · +85 −6" · "seq 01–03 / 05" · "conf 0.72". Tabular numerals. Zero-pad sequence numbers.
7. **Empty states are victories.** "queue clear — everything is forwarding at line rate." Never apologize for emptiness.
8. **Explain with one clause, not a paragraph.** Why-held strings are single lines: "Handshake is example-only on a strict lane · conf 0.72".
9. **Links point with a trailing →** and name the destination: "open the real inspector →", "control room →".
10. **The dry aside.** One ✱-prefixed sentence per screen may editorialize: "Everything not on this list forwarded without you — that's the point."

## Sample strings (canonical)

- "214 packets forwarded without you today"
- "the machine at work — nothing to do here"
- "green means gone"
- "looking is cheap"
- "held 34m · handshake below lane floor"
- "never trusted · always verified before prod"
- "lands friday 09:00 · nothing here can interrupt you"

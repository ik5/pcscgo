# Smart Card Basics

A from-zero guide to smart cards, readers, the ATR, the T=0/T=1 protocols,
and the PC/SC API — written for a Go developer who has never touched a
smart card. After this document, `examples/scan` and the `pcscgo` API
reference should read like familiar territory.

> Scope: contact cards on Linux/BSD via PC/SC-Lite (what `pcscgo`
> binds to). Contactless (NFC) and Windows/macOS specifics are out of scope.

---

## 1. The cast of characters

### 1.1 The card

A smart card is a small computer: a microcontroller (CPU, RAM, ROM/EEPROM
or flash) embedded in plastic, with metal **contacts** on the surface.
Unlike a magnetic stripe or a plain memory chip, the chip runs its own
operating system and refuses to hand over secrets (keys, PINs) directly —
you can only ask it to *do things* (verify a PIN, sign data, return a file)
by sending commands.

Two families exist:

- **Memory cards** (e.g. old phone cards) — dumb storage, no CPU. Rare
  today; PC/SC tooling mostly ignores them.
- **Microprocessor cards** (SIM/USIM, EMV bank cards, ID cards, PIV/CAC)
  — a real CPU. Everything in this guide assumes these.

A SIM card is just a microprocessor card in a small package. The "sizes"
(nano/micro/mini) are the **same chip with different amounts of plastic
around it** — an adapter board is electrically passive, it only repositions
the contacts. Keep that in mind in the troubleshooting section ([§7 Troubleshooting: the adapter-board scenario](#7-troubleshooting-the-adapter-board-scenario)):
an adapter cannot change what the chip says, but it *can* break the
electrical connection, and a broken connection looks like
`SCardStateMute` (present but not answering).

Cards speak over the contacts defined by ISO 7816-2: power (VCC), reset
(RST), clock (CLK), ground (GND), programming voltage (VPP, obsolete), and
one serial data line (I/O). That **one I/O contact** carries bytes both
ways, not at the same time (half-duplex). There is no separate TX and RX
wire. T=0 and T=1 are protocols on that single line
([§8 T=0 vs T=1: the two transport protocols](#8-t0-vs-t1-the-two-transport-protocols)).

### 1.2 The reader

The reader is the electrical bridge between USB and the card contacts. Most
USB readers speak **CCID** (Chip Card Interface Device), a standard USB
class — that is why one generic driver covers most readers. The reader:

- supplies power (modern cards: 1.8 V, 3 V, or 5 V — the reader negotiates the class during activation, [§10.1 Voltage is the reader's job, not yours](#101-voltage-is-the-readers-job-not-yours)),
- supplies the clock,
- performs the **reset sequence** that makes the card emit its ATR ([§5 ATR anatomy, byte by byte](#5-atr-anatomy-byte-by-byte)),
- shuffles bytes between USB and the card's I/O line.

The reader does **not** understand your application. It is a pipe with
power management.

### 1.3 The driver, the daemon, and the library

On Linux/BSD the stack has four layers:

```
Your Go app
    │  pcscgo API (SCardEstablishContext, SCardConnect, SCardTransmit, …)
    ▼
libpcsclite (C library — speaks to the daemon over a Unix socket)
    ▼
pcscd (daemon — owns the readers, multiplexes apps, runs hot-plug)
    ▼
IFD handler / CCID driver (translates to USB packets for your reader)
    ▼
Reader hardware ──► Card
```

- **pcscd** (the "resource manager") must be **running**
  (`systemctl start pcscd`). If it isn't, every call fails with
  `SCardErrorNoService` (0x8010001D). It is the only process that talks
  to readers; all apps share it.
- **libpcsclite** is the C client library. `pcscgo` does not reimplement
  any of this stack — it is a thin translation layer from Go to
  libpcsclite (see `README.md` "How pcscgo Fits In").
- **Hot-plug**: pcscd watches USB. Reader arrival/removal is reported
  through a pseudo-reader named `\\?PnP?\Notification` ([§6.4 Reader add/remove: the PnP entry](#64-reader-addremove-the-pnp-entry)).

---

## 2. The PC/SC session model

Every PC/SC program follows the same six steps. The `pcscgo` names are
the C names with Go error returns:

| Step | `pcscgo` call | What it means |
|------|---------------|---------------|
| 1. Open a session with pcscd | `SCardEstablishContext(SCardScopeUser, nil, nil, &ctx)` | "I want to use smart cards." Returns a context handle. |
| 2. Find readers | `SCardListReaders(ctx, "", buf)` | Returns a **multi-string**: names separated by `\x00`, terminated by `\x00\x00`. Call once with an empty buffer to learn the size, allocate, call again. `SCardErrorNoReadersAvailable` means "zero readers" — treat as an empty list, not a failure. |
| 3. Connect to a card | `SCardConnect(ctx, name, SCardShareShared, SCardProtocolAny, &card, &proto)` | Powers the card if needed, returns a card handle and the negotiated protocol (T=0 or T=1). Share mode is usually `SCardShareShared`. |
| 4. Exchange commands | `SCardTransmit(card, pci, send, recvPci, recv)` | Send one command APDU, get the response ([§8 T=0 vs T=1: the two transport protocols](#8-t0-vs-t1-the-two-transport-protocols)). `pci` must match the negotiated protocol: `SCardPCIT0()` or `SCardPCIT1()`. |
| 5. Disconnect | `SCardDisconnect(card, SCardLeaveCard)` | `SCardLeaveCard` = leave the card powered; `SCardUnpowerCard` = power it down (next use re-reads the ATR). |
| 6. Close the session | `SCardReleaseContext(ctx)` | |

Monitoring (watching for insert/remove without polling in a tight loop)
uses a separate call, `SCardGetStatusChange`, covered in [§6 Detecting what happened: insert, remove, disconnect, mute](#6-detecting-what-happened-insert-remove-disconnect-mute).

Key vocabulary:

- **Context** — your session with pcscd.
- **Reader** — a named slot managed by pcscd (one USB reader can expose
  several slots).
- **Card handle** — your connection to one card in one reader.
- **APDU** (Application Protocol Data Unit) — one command/response pair.
- **ATR** (Answer to Reset) — the bytes the card sends when powered
  and reset. The subject of [§5 ATR anatomy, byte by byte](#5-atr-anatomy-byte-by-byte).

### The card file system (four terms)

`SCardConnect` opens a session with the chip. It does **not** choose a
file. Inside the chip, data lives in a small tree. Four words:

- **MF** — Master File, the root.
- **DF** — Dedicated File, a directory. An application is usually a DF.
- **EF** — Elementary File, a leaf that holds bytes (one blob, or records).
- **AID** — Application Identifier. **SELECT** (INS `0xA4`) chooses a DF
  or EF by path, file ID, or AID.

```
MF                      <- root
+-- DF  (one application)
|   +-- EF              <- data (blob or records)
|   `-- EF
`-- DF  (another application)
    `-- EF
```

`SCardConnect` is a session with the chip. SELECT sets the current file.
Until SELECT succeeds, the card has no current file. Reads and writes
then often return `69 86` or `6A 82`. That is the card talking, not a
PC/SC reader failure. The same four terms apply to any microprocessor
card; which DFs and EFs exist is family-specific (telecom, payment, ID).

---

## 3. APDUs: structure, framing, validation, recovery

Everything after the ATR is APDUs — command/response pairs, identical on
every card family. This section is implementation-agnostic: no SIM or EMV
specifics, just the wire grammar, how to parse it, how to know a response
is complete, and how to recover from the standard failures.

### 3.1 Command anatomy: identify the case from the bytes

Every command starts with a 4-byte header:

```
CLA  INS  P1  P2  [Lc] [Data…] [Le]
```

- `CLA` — instruction class (`0x00` plain, `0x0C` secure messaging).
- `INS` — the instruction (`0xA4` = SELECT, `0xB0` = READ BINARY, …).
- `P1 P2` — instruction parameters (offsets, IDs, references).
- `Lc` — length of outgoing `Data`; `Le` — max response bytes wanted.

The body's shape falls into four cases — identified purely by length:

```
Case 1:  CLA INS P1 P2
Case 2:  CLA INS P1 P2  Le
Case 3:  CLA INS P1 P2  Lc  Data...
Case 4:  CLA INS P1 P2  Lc  Data...  Le
```

Each box is one byte except `Data...`. `Lc` is the length of `Data`;
`Le` is how many response bytes you will accept.

| Case | Layout | Total length | Example |
|------|--------|--------------|---------|
| 1 | header only | 4 | `00 A4 00 00` (header only, not a complete SELECT-with-AID) |
| 2S | header + `Le` | 5 | `… Le` — last byte is Le |
| 3S | header + `Lc` + Data | 5 + Lc | byte[4] = Lc = len − 5 |
| 4S | header + `Lc` + Data + `Le` | 6 + Lc | byte[4] = Lc = len − 6 |

(Short `Le`: one byte, `0x00` means 256. Extended length inserts a
`00` marker and uses two-byte lengths — same four cases, bigger
numbers; needs T=1 or extended-capable T=0 plus large buffers,
[§8.3 What this means for your code](#83-what-this-means-for-your-code).) The parse rule: read byte[4] as a *candidate* Lc — if the total
length equals 5+Lc it's case 3, if 6+Lc it's case 4, if the command is
exactly 5 bytes long the last byte is Le (case 2), exactly 4 is case 1.
Anything else is malformed before it ever reaches the card.

### 3.2 Response framing: knowing it ended

`SCardTransmit` returns `n`, and `recv[:n]` is laid out as:

```
[response data…] SW1 SW2
```

The **last two bytes are always the status word**; everything before them
is data (possibly zero bytes). `n` includes the status word, so a "read
10 bytes" answer arrives as n=12. If you asked for more than the card
has, it returns what it has plus `90 00` (or `61 xx` on T=0 — [§3.4 Recovery: the three resend patterns](#34-recovery-the-three-resend-patterns)).
Framing discipline: always split `data = recv[:n-2]`, `sw = recv[n-2:]`;
`n < 2` is a transport failure, not a card answer.

### 3.3 Validation: reading the status word

| SW | Class | Meaning |
|----|-------|---------|
| `90 00` | Success | Done — data (if any) is valid |
| `61 xx` | More available (T=0) | `xx` bytes waiting — fetch with GET RESPONSE ([§3.4 Recovery: the three resend patterns](#34-recovery-the-three-resend-patterns)), not an error |
| `6C xx` | Wrong length | Resend the identical command with `Le = xx` ([§3.4 Recovery: the three resend patterns](#34-recovery-the-three-resend-patterns)), not an error |
| `63 Cx` | Counter | PIN-style retry counter, X tries left ([§10.3 PINs: when, which, how](#103-pins-when-which-how)) |
| `67 00` | Wrong length | Lc/Le don't match the body — recount ([§3.1 Command anatomy: identify the case from the bytes](#31-command-anatomy-identify-the-case-from-the-bytes)), your bug |
| `68 xx` | CLA problems | e.g. wrong class / logical channel |
| `69 82/83/86` | Security / selection | PIN needed / blocked / nothing selected ([§10.2 "Can I write to this file?"](#102-can-i-write-to-this-file)) |
| `6A 80/82/86` | Data / params | Bad data / file not found / wrong P1 P2 |
| `6D 00` / `6E 00` | Unknown INS / CLA | Card doesn't know this instruction — wrong app selected, or wrong card family |

### 3.4 Recovery: the three resend patterns

1. **`61 xx` → GET RESPONSE loop.** Send `00 C0 00 00 xx`, append its
   data, repeat while the answer is again `61 yy`, until `90 00`. (Mostly
   T=0; T=1 chains below your APDU instead.)
2. **`6C xx` → resend with corrected Le.** Identical header, identical
   data, last byte = `xx`.
3. **Timeout / no response → verify before retrying a write.** An
   unanswered command *may still have executed* on the card. Re-issue
   blindly and a debit/verify-counter/increment can happen twice. For
   writes, follow a timeout with a status *read* first; only retry the
   write if the read proves it didn't land. Reads are safe to re-issue
   freely.

```
61 xx  (more data, mostly T=0)
  host --> COMMAND
  card <-- 61 xx
  host --> 00 C0 00 00 xx     GET RESPONSE
  card <-- data + 61 yy   or  data + 90 00
           (repeat GET RESPONSE until 90 00)

6C xx  (wrong Le)
  host --> COMMAND  (Le wrong)
  card <-- 6C xx
  host --> COMMAND  (same header+data, Le=xx)
  card <-- data + 90 00
```

What `pcscgo` does and doesn't do here: `SCardTransmit` transports one
APDU and reports `n` + error. The `61`/`6C` loops, buffer sizing
(`pcscgo.MaxBufferSize` = 264 for short APDUs,
`pcscgo.MaxBufferSizeExtended` for extended), and timeout discipline are
caller code — [§8.3 What this means for your code](#83-what-this-means-for-your-code).

You never parse voltages or clocks yourself: `SCardTransmit` takes the
byte slice, the reader and card do the rest.

---

## 4. Power-up: reset, ATR, and PPS

When a card is powered (or `SCardReconnect` with reset, or a `SCardConnect`
to an unpowered card), this happens electrically:

1. RST is held low. The reader applies VCC, then CLK.
2. The reader **releases** RST (the line goes inactive/high). It does not
   send a short pulse.
3. The card answers on the I/O line with the **ATR** — 2 to 33 bytes.
4. Optionally, reader and card negotiate faster/different parameters
   (**PPS** — Protocol and Parameters Selection). If PPS fails, they fall
   back to the ATR's defaults.

So the ATR is the card's self-description, spoken once per reset, *before*
any command is exchanged. Its jobs: announce timing parameters, announce
which protocols (T=0/T=1) the card supports, and carry free-form
"historical" bytes the issuer uses for identification.

Two ATR flavors exist: the **cold ATR** (after power-up) and the **warm
ATR** (after a reset without power-down). They can legitimately differ —
usually only in the historical bytes.

---

## 5. ATR anatomy, byte by byte

Notation below: `T0`, `TA1`… are the standard names from ISO 7816-3.
`pcscgo` hands you the raw ATR bytes (e.g. in
`SCardReaderState.ATR[:ATRLength]`); parsing them is your code's job.

Structure:

```
TS | T0 | [TA1] [TB1] [TC1] [TD1] | [TA2] ... | [H1..Hk] | [TCK]
     |
     +-- T0 splits as Y nibble (flags) and K nibble (hist count)
```

### 5.1 TS — the very first byte: which bit convention?

`TS` is always exactly one of two values:

- `0x3B` — **direct convention**: bits are sent as-is. This is the usual TS.
- `0x3F` — **inverse convention**: every byte is transformed (see below).
  Rare; old cards.

#### What "inverse" actually means

Two things are flipped at once, per byte:

1. **Voltage polarity**: low voltage means logic 1 (instead of high = 1).
2. **Bit order**: the most significant bit goes out first (instead of
   least significant bit first).

Concretely: if an inverse-convention card sends its TS and you sample it
with direct-convention assumptions, you read `0x03` (`0000 0011`).
Reverse the order of all 8 bits (`1100 0000` = `0xC0`) and flip every bit
(`0011 1111`) and you get `0x3F` — the inverse marker. The same
reverse-plus-invert transform recovers every following byte:

```go
// reverseBits mirrors bit order inside one byte.
func reverseBits(b byte) byte {
	var r byte
	for i := 0; i < 8; i++ {
		if b&(1<<i) != 0 {
			r |= 1 << (7 - i)
		}
	}
	return r
}

// inverseDecode recovers the true byte value from a wire byte sampled
// under direct-convention assumptions: flip polarity, flip bit order.
func inverseDecode(wire byte) byte { return ^reverseBits(wire) }

fmt.Printf("%02X\n", inverseDecode(0x03)) // 3F — inverse marker confirmed
```

#### Is this endianness? No.

Endianness is the order of *bytes* inside a multi-byte word (e.g. whether
a `uint16` stores its high byte first). The ATR is a flat byte stream —
there are no multi-byte words in it, so endianness never comes up. What is
flipped here is the order of *bits within each single byte*, plus the
voltage meaning of 0/1. In practice you never implement this yourself:
the reader hardware detects `3B` vs `3F` during the reset sequence and
decodes accordingly, so PC/SC always hands you finished byte values.
You only need to know the transform exists so a `3F` first byte doesn't
look like corruption.

### 5.2 T0 — the table of contents

`T0` packs two things into one byte:

```
          Y = T0 >> 4                 K = T0 & 0x0F
T0:  [ TD | TC | TB | TA ]       [ historical count 0..15 ]
mask:  8    4    2    1
raw: 0x80 0x40 0x20 0x10         T0&0x1 is K, not TA1
```

Example: `T0 = 0x9F` → Y=`9` (`1001`) → TA1 and TD1 present; K=15.

- **High nibble** — presence flags for the first group of interface
  bytes, consumed in the order TA1, TB1, TC1, TD1. Take `y := T0 >> 4`,
  then: `y&0x1` → TA1, `y&0x2` → TB1, `y&0x4` → TC1, `y&0x8` → TD1.
  On the **raw** T0 byte those same flags are `0x10`/`0x20`/`0x40`/`0x80`;
  `T0&0x1` is *not* TA1 (that bit belongs to K). Every `TDi` reuses this
  nibble scheme for the next group (`0x1`→TA, `0x2`→TB, `0x4`→TC,
  `0x8`→TD).
- **Low nibble (bits 4–1)** — `K`, the number of **historical bytes**
  (0–15). The historical bytes are a free-form identification block at
  the tail of the ATR (issuer data, version strings, TLV objects) —
  what lives inside them, and how the first one tells you its own
  format, is described in
  [§5.5 Historical bytes — the free-form ID card](#55-historical-bytes--the-free-form-id-card).

### 5.3 TA1, TB1, TC1 — first parameter group

These appear in the order TA1, TB1, TC1, TD1 — only the ones flagged in T0.

- **TA1 — clock and speed.** High nibble `FI` (clock-rate conversion
  factor), low nibble `DI` (baud-rate adjustment factor). One bit time
  ("etu") = `F/D` clock cycles. The default `TA1 = 0x11` means F=372,
  D=1 → 372 clocks per bit (~9600 bit/s at 3.57 MHz). Smaller F or larger
  D = faster. Common values:

  | TA1 | F | D | clocks/bit | vs default |
  |-----|---|---|------------|------------|
  | `0x11` | 372 | 1 | 372 | default |
  | `0x95` | 512 | 16 | 32 | ~12× faster |
  | `0x96` | 512 | 32 | 16 | ~23× faster (common fast-clock TA1) |

  The multiplier is `372 / (F/D)`, not D. D=16 is the baud factor, not
  "16× speed". (Full FI/DI tables are in ISO 7816-3; these three cover
  nearly every card you will meet.)

- **TB1 — programming voltage (VPP).** Obsolete — modern cards generate
  programming voltage internally and ignore VPP. `TB1 = 0x00` means "no
  VPP needed", and the standard now says cards should omit TB1 and
  readers should ignore it (EMV still requires `TB1 = 0x00` to be sent).
  You will essentially never act on this.
- **TC1 — extra guard time.** `N`, the number of extra bit-times the card
  needs between bytes. `0x00` (or absent) = default (12 etu for T=0).

### 5.4 TD1, TD2, … — protocol announcements and the chain

Each `TDi` byte also packs two things:

- **High nibble** — presence flags for the *next* group, same scheme as
  T0 and same consumption order (TA, then TB, then TC, then TD):
  `0x1` → `TA(i+1)`, `0x2` → `TB(i+1)`, `0x4` → `TC(i+1)`,
  `0x8` → `TD(i+1)`.
- **Low nibble** — the protocol this group describes: `0` = T=0,
  `1` = T=1, `15` (`0xF`) = "global parameters, no new protocol".

The chain ends at the first `TDi` whose high nibble is `0`. If *no* TD
byte appears at all, the card offers **T=0 only**.

- **TA2 — mode.** Present only when some TD flags it. Bit 8 (`0x80`):
  `0` = **specific mode** (use exactly these parameters, no PPS);
  `1` = **negotiable mode** (PPS may change them). Bit 5 (`0x10`):
  `0` = use the interface-byte parameters, `1` = implicit defaults.
  Bits 4–1 repeat the protocol number.
- **TC2 (T=0)** — work waiting time (`WI`): how long the card may take
  before answering.
- **TA3/TB3 (T=1)** — block-size and timing tuning (`IFSC`, `BWI/CWI`).
  Defaults work; you only care if you tune T=1 performance.
- **TCi for T≠0** — usually `0x00`/absent; link-layer details.

### 5.5 Historical bytes — the free-form ID card

The interface bytes (TA/TB/TC/TD) answer "how do we talk?" — speed,
protocol, extra guard time. The **historical bytes** answer "who is this
chip?" They are `K` extra bytes (`K` = T0's low nibble, 0–15,
[§5.2 T0 — the table of contents](#52-t0--the-table-of-contents)) sitting
**after** the last interface byte and **before** the optional TCK.

They are not commands, not file contents, and not voltage. No APDU has
run yet. The issuer stuffed a short identity blob into the reset answer
so a host can recognise the card (or at least its family) before
`SELECT`. PC/SC just hands you the raw ATR; parsing this blob is your
code.

**What they are not a substitute for:** a historical AID-like field, if
present, is a hint. The real application list still comes from `SELECT`
/ scanning after you connect.

The first historical byte is a **category indicator** (ISO 7816-4). It
says how to read the remaining `K−1` bytes:

- `0x00` — the **last three** historical bytes are a status indicator:
  one life-cycle byte plus two status bytes (the same shape as SW1 SW2).
  Life-cycle `0x00` means **no information given**, not "powered".
- `0x80` — the rest is **compact-TLV** (ISO 7816-4): each object is one
  header byte (high nibble = tag, low nibble = length) followed by that
  many value bytes. This is not PC/SC Part 10 and not
  `PCSCTLVStructure` (those describe reader features, a different TLV
  on a different path).
- Any other first byte — **proprietary**. Do not treat it as an
  application identifier unless that card's specification says so.

Card-type identification by ATR ([§5.8 Practical ATR use](#58-practical-atr-use))
mostly matches a prefix covering TS…TD plus the first historical bytes.

### 5.6 TCK — the checksum

Present **if and only if** some TD byte announced a non-zero protocol
(i.e. T=1 or higher — in practice: T=1 offered). Value = XOR of **all**
bytes from `T0` onward (TS excluded). If your parser computes the XOR and
it mismatches, the ATR was corrupted on the wire — suspect contacts ([§7 Troubleshooting: the adapter-board scenario](#7-troubleshooting-the-adapter-board-scenario)).

When there is **no TCK** (T=0 only — no TD announced T≠0), error handling
moves down a layer rather than disappearing. T=0 protects every byte
individually: each character carries an even-parity bit, and a receiver
that sees bad parity pulls the I/O line low to demand retransmission of
that one byte — corruption is retried on the spot instead of detected
after the fact. (T=1 does the same job per block with its LRC/EDC
checksum plus R-block retries, [§8.2 T=1 — block-oriented](#82-t1--block-oriented).) What no-TCK costs you is a
whole-message integrity verdict: with TCK you can prove the full ATR
arrived intact; without it, validate structurally instead — walk the
flag chain ([§5.7 Worked example (constructed, self-consistent)](#57-worked-example-constructed-self-consistent)) and confirm it consumes exactly K historical bytes with
nothing left over. A walk that overruns, truncates, or leaves stray bytes
is the T=0-only equivalent of a checksum failure: treat the ATR as
suspect and re-read ([§7.3 Checksum mismatch: what to do](#73-checksum-mismatch-what-to-do)).

### 5.7 Worked example (constructed, self-consistent)

ATR: `3B F0 11 00 00 11 01 F1`

| Byte | Decode |
|------|--------|
| `3B` | TS: direct convention |
| `F0` | T0: `TA1,TB1,TC1,TD1` present (`F`=1111), `K`=0 historical bytes |
| `11` | TA1: F=372, D=1 — default speed |
| `00` | TB1: no programming voltage |
| `00` | TC1: default guard time |
| `11` | TD1: next group has TA2 only (`1`=0001 → TA); protocol nibble `1` → **T=1 offered** (T=0 is always implicit) |
| `01` | TA2: specific mode (bit 8 = 0), protocol `1` → T=1; TA2 is not a TD byte, so the chain ends here |
| `F1` | TCK: `F0^11^00^00^11^01 = F1` ✓ (present because T=1 was offered) |

Real card ATRs are longer (historical bytes, more groups), but every one
parses with exactly this procedure: read TS, read T0, walk the flagged
groups, read K historical bytes, check TCK if any TD announced T≠0.

#### The same decode as Go code

Each table row above is one small bit operation. T0 first:

```go
t0 := atr[1]
y := t0 >> 4   // high nibble: presence flags
k := t0 & 0x0F // low nibble: historical byte count
hasTA1 := y&0x1 != 0
hasTB1 := y&0x2 != 0
hasTC1 := y&0x4 != 0
hasTD1 := y&0x8 != 0
// t0 = 0xF0 -> y = 0xF (all four present), k = 0
```

TA1 splits the same way; the nibbles index the F/D tables:

```go
ta1 := atr[2]
fi, di := ta1>>4, ta1&0x0F
fTable := map[byte]int{0x1: 372, 0x9: 512, 0xA: 768, 0xB: 1024}
dTable := map[byte]int{0x1: 1, 0x5: 16, 0x6: 32}
etu := float64(fTable[fi]) / float64(dTable[di]) // clocks per bit
// ta1 = 0x11 -> FI=1 (F=372), DI=1 (D=1) -> 372 clocks/bit (default)
// ta1 = 0x95 -> FI=9 (F=512), DI=5 (D=16) -> 32 clocks/bit (~12x faster)
```

Each TD byte: high nibble continues the chain, low nibble names the protocol:

```go
td1 := atr[5]
nextY, proto := td1>>4, td1&0x0F
// td1 = 0x11 -> next group has TA2 only, protocol 1 (T=1)
ta2 := atr[6]
// ta2 = 0x01 -> specific mode, T=1; TA2 is not a TD, so the chain ends
```

TCK re-computed over everything from T0 up to (excluding) itself:

```go
var x byte
for _, b := range atr[1 : len(atr)-1] {
	x ^= b
}
match := x == atr[len(atr)-1] // must hold; mismatch = line noise
```

Running the full sequence on the example ATR prints (verified by execution):

```
direct convention: true
T0=F0 -> Y=F K=0
TA1 present: true
TB1 present: true
TC1 present: true
TD1 present: true
TA1=11 -> FI=1 DI=1
TD1=11 -> next Y=1 protocol=1
TA2=01 -> specific mode, protocol 1, chain ends
TCK=F1 computed=F1 match=true
```

### 5.8 Practical ATR use

1. **Card-type identification**: match the ATR (or its prefix) against
   known values — SIM/USIM/ISIM/EMV each have characteristic prefixes.
2. **Protocol choice**: the TD chain tells you whether T=1 is available;
   `SCardConnect` with `SCardProtocolAny` negotiates it, and the
   `activeProtocol` out-parameter tells you what won. Pass the matching
   PCI (`SCardPCIT0()` / `SCardPCIT1()`) to `SCardTransmit`.
3. **Speed**: TA1 tells you the negotiated bit rate (informational at the
   PC/SC level — the reader applies it).
4. **Sanity limits**: an ATR is at most 33 bytes (`pcscgo.MaxATRSize`).
   Zero-length ATR + card-present state = the card isn't answering ([§7 Troubleshooting: the adapter-board scenario](#7-troubleshooting-the-adapter-board-scenario)).

### 5.9 Case study: one card, two ATRs

The two readings below come from the same card in the same reader — once
seated directly, once via a passive multi-size adapter. Both were
machine-parsed with the procedure from [§5.7 Worked example (constructed, self-consistent)](#57-worked-example-constructed-self-consistent) and the checksums recomputed.
This is what "reading an ATR" looks like in practice, and what adapter
trouble looks like next to a healthy baseline.

**Direct (no adapter):**
`3B 8F 80 01 80 4F 0C A0 00 00 03 06 03 00 01 00 00 00 00 6A`

| Bytes | Decode |
|-------|--------|
| `3B` | TS: direct convention |
| `8F` | T0: TD1 only (`8`), K=15 historical bytes |
| `80` | TD1: TD2 follows (`8`); protocol T=0 |
| `01` | TD2: chain ends; protocol T=1 offered → TCK present |
| `80 4F 0C …` (15 bytes) | Historical block (see below) |
| `6A` | TCK: recomputed XOR = `6A` ✓ |

The historical block starts with category `80` ([§5.5 Historical bytes — the free-form ID card](#55-historical-bytes--the-free-form-id-card)).
What follows is **not** compact-TLV (tag and length would share one
byte). It is BER-TLV: tag `4F` (application identifier), length `0C`,
value `A0 00 00 03 06 03 00 01 00 00 00 00`. That `A0 00 00 …` RID is a
common PC/SC SIM/storage-card prefix — confirm against the issuer's
documentation. No TA1, so default speed (372 clocks/bit). Structure and
TCK both check out. **This is the reference reading.**

**Via multi-size adapter:**
`3B 9F 96 80 1F 87 80 31 E0 73 FE 21 19 67 4C 43 54 30 31 01 4A 83`

| Bytes | Decode |
|-------|--------|
| `3B` | TS: direct convention |
| `9F` | T0: TA1 + TD1 (`9`), K=15 |
| `96` | TA1: F=512, D=32 — claims ~23× default speed |
| `80` | TD1: TD2 follows; protocol T=0 |
| `1F` | TD2: TA3 follows (`1` → TA); `F` = T=15 global, not a transport protocol |
| `87` | TA3 (global); chain ends (TA3 is not a TD byte) |
| 15 hist bytes | Starts `80 31 E0 …` — see below |
| `83` | TCK: T=15 was indicated, so TCK **must** be present; XOR of T0..hist = `83` ✓ |

The checksum is valid: the reader received these bytes intact. Wire
noise would have broken TCK. What is wrong is the **content**:

- After category `80`, the remaining 14 bytes do **not** parse as
  compact-TLV (the next objects overrun or leave a broken tail).
- The ASCII fragment `LCT01` (`4C 43 54 30 31`) sits in that tail —
  issuer-string debris in a block that does not parse.
- `TA1=96` claims a fast clock while the direct reading used defaults:
  same chip, different negotiation path.

Valid TCK + unparseable historical bytes means the card emitted a
confused identity blob (typical of a messy reset through an adapter),
not that a byte was dropped on the wire. Do not match this ATR against
card-type tables, and do not trust the TA1 speed. **Action: reseat the
adapter, read 2–3 times, and only trust ATRs that validate end to end
like the direct reading.**

---

## 6. Detecting what happened: insert, remove, disconnect, mute

The ATR never tells you about *events*. Events come from
**`SCardGetStatusChange`**, which blocks until something changes and fills
in each entry's `EventState` (and ATR). The mental model:

- `CurrentState` = what you last saw. `EventState` = what is true now.
- You say "I last saw X, wake me when it differs", then copy
  `EventState → CurrentState` and repeat (`examples/scan` does exactly this).
- Initialize with `SCardStateUnaware` ("I know nothing yet").

### 6.1 The state flags that matter

(From `pcsclite.go`; full list with hex values is there.)

| Flag | Meaning |
|------|---------|
| `SCardStatePresent` | A card is in the reader |
| `SCardStateEmpty` | Reader is empty |
| `SCardStateMute` | Card is present but **not answering** (no usable ATR) |
| `SCardStateUnpowered` | Card present but not powered |
| `SCardStateUnavailable` | The **reader** is gone/unusable (not the card) |
| `SCardStateChanged` | "Something changed" — see [§6.2 The three traps](#62-the-three-traps), never trust it alone |
| `SCardStateIgnore` | Skip this reader in this call |
| `SCardStateExclusive` / `InUse` | Another app holds the card exclusively / shared |

### 6.2 The three traps

1. **The `Changed` flag alone means nothing.** Compare full states
   *ignoring* that bit, exactly like `examples/scan`:
   ```go
   if state.EventState&^pcscgo.SCardStateChanged == state.CurrentState&^pcscgo.SCardStateChanged {
       return // No change — spurious wakeup
   }
   ```
2. **The upper 16 bits of `EventState` are an event counter**, not flags.
   pcscd increments it on every insert/remove, so two readings like
   `0x022` then `0x10022` are the *same flags* with a new event. Always
   mask with `0xFFFF` (or compare flag-by-flag) before interpreting.
3. **`SCardErrorNoReadersAvailable` is an empty list**, not an error —
   the last reader was unplugged between your calls. Re-enumerate.

### 6.3 Reading the four situations

| Observation | Interpretation |
|-------------|----------------|
| `Empty → Present`, ATR present, length > 0 | **Card inserted**, healthy. Read the ATR ([§5 ATR anatomy, byte by byte](#5-atr-anatomy-byte-by-byte)). |
| `Present → Empty` | **Card removed** (pulled out). |
| `Present` + **`Mute`** set, ATR length 0 or garbage | Card is electrically there but **not communicating**: dirty/misaligned contacts, adapter seating ([§7 Troubleshooting: the adapter-board scenario](#7-troubleshooting-the-adapter-board-scenario)), wrong voltage, or a dead chip. This is the "disconnected but not removed" case. |
| `Present` + `Unpowered` | Powered down (e.g. after `SCardDisconnect(…, SCardUnpowerCard)`). Reconnect to power it up. |
| `Unavailable` set/cleared | The **reader** (USB device) came or went — not a card event at all. |
| `Present`, ATR parses but *differs* from last time for the same card | Re-read, not a new card: warm-vs-cold ATR, a different reader's PPS result ([§7.3 Checksum mismatch: what to do](#73-checksum-mismatch-what-to-do)), or line noise. Compare TS/T0/TD (protocol + params); if only historical bytes differ, it is almost certainly the same card. |

### 6.4 Reader add/remove: the PnP entry

Append one extra `SCardReaderState` with reader name
`"\\\\?PnP?\\Notification"` (as `examples/scan` `setupStates` does).
pcscd sets only `Changed` (+ counter) on it. That event means "the reader
*list* may have changed" — the actual added/removed names come from
calling `SCardListReaders` again and diffing (`rebuildIfNeeded` in the
example). Never infer *which* reader changed from the PnP entry itself.

---

## 7. Troubleshooting: the adapter-board scenario

Two common field observations, decoded with the model above:

**Case A — card seated directly: short/basic ATR, "nothing more."**
A minimal ATR (TS+T0, few or no interface bytes, short history) only means
the card negotiates the *defaults*: T=0 at 372 clocks/etu. "Nothing more"
afterward is an application-layer matter — connecting and selecting an
application (SELECT by AID) is a separate step from the ATR, and a bare
ATR never lists files or keys. If even `SCardConnect` fails, suspect power
or sharing (another process holding the card exclusive), not the ATR.

**Case B — multi-size adapter: the reader enumerates, the slot has no usable ATR.**
The *reader* enumerates (PnP, name, slots) but the *slot* reports
`Present`+`Mute`, or `Present` with
`ATRLength == 0`. The passive adapter did not change the chip's answer —
it degraded the contacts (alignment, pressure, thickness, dirt, worn
adapter springs). The card is electrically "there" (presence detect is a
mechanical switch) but the I/O line can't carry a clean reset sequence.

Checklist, in order:

1. Run `examples/scan -v` and read the flags, not just the ATR line:
   `Mute` → electrical; `Unpowered` → power state; `Unavailable` → USB/reader.
2. Reseat the card in the adapter, then the adapter in the reader; try a
   different adapter. SIM adapters are notorious for micro-misalignment.
3. Clean contacts (dry lint-free cloth; no abrasives).
4. Try the card direct again in the same reader — if direct works and
   adapted doesn't, the adapter is the whole story.
5. Try another reader (rules out voltage/clock quirks of one reader).
6. Check `pcscd` logs (`pcscd --foreground --debug`) for PPS failures or
   "card mute" lines — that distinguishes negotiation failure from silence.
7. Verify TCK ([§5.6 TCK — the checksum](#56-tck--the-checksum)) on any ATR you *do* get: a mismatch means line noise,
   i.e. marginal contacts, even when a parseable ATR arrives.

### 7.1 Flags → action cheat sheet

Read the flags ([§6.1 The state flags that matter](#61-the-state-flags-that-matter)–[§6.2 The three traps](#62-the-three-traps)) first — remember the two mechanical rules:
ignore the `Changed` bit when comparing states (a set `Changed` bit with
otherwise identical states is a spurious wakeup), and mask off the upper
16 bits of `EventState` (that's pcscd's insert/remove counter, not flags).
Then:

| What you see | Meaning | What to do |
|--------------|---------|------------|
| `Empty → Present`, ATR validates ([§5.9 Case study: one card, two ATRs](#59-case-study-one-card-two-atrs)) | Healthy insert | Parse ATR, `SCardConnect`, proceed |
| `Present` + `Mute`, ATR length 0 | Card silent: contacts, seating, voltage, or dead chip | Reseat → clean → try direct (no adapter) → try another reader → read pcscd debug logs |
| `Present`, ATR fails structure/TCK | Confused card or noisy line ([§5.9 Case study: one card, two ATRs](#59-case-study-one-card-two-atrs) adapter case) | Re-read 2–3 times; trust only stable, validating ATRs |
| `Present` + `Unpowered` | Powered down (e.g. after disconnect-with-unpower) | `SCardConnect`/`SCardReconnect` to power it back up |
| `Unavailable` toggles | The USB reader came/went — not a card event | Re-enumerate with `SCardListReaders`, diff, rebuild states |
| PnP entry `Changed` only | Reader list *may* have changed | `SCardListReaders` again and diff — never infer which reader from the PnP entry |
| `Changed` set, masked states equal | Spurious wakeup | Ignore, keep waiting |

### 7.2 Why can the "same card" show different ATRs?

Candidate causes, easiest to test first:

- **Different reader/path** (direct vs adapter, or reader A vs B):
  PPS negotiation results differ, and some readers report the
  post-PPS parameters while others report the cold ATR.
- **Cold vs warm ATR**: power-up vs reset-without-power-down; historical
  bytes may differ.
- **Marginal contacts**: a corrupted ATR fails TCK or parses oddly —
  flaky, changes between reads.
- **Specific mode (TA2)**: a card in specific mode locks parameters;
  readers that attempt PPS anyway can report different views.

Test: compare the two ATRs field by field ([§5.7 Worked example (constructed, self-consistent)](#57-worked-example-constructed-self-consistent)). Same TS/T0/TD with
different history = same card, different reset context. Different TD
chain = different negotiation path.

### 7.3 Checksum mismatch: what to do

A TCK mismatch means bytes were corrupted **in transit** between card
and reader — contact noise, clock jitter, a loose adapter. The content
is untrustworthy: do not cache it, do not base PPS speed negotiation on
its TA1, do not match it against card-type tables. Steps:

1. **Re-read immediately.** Single-event noise is common; a second clean
   read ends the story.
2. **If it repeats**, work the seating ladder: reseat the card, clean the
   contacts, drop any adapter, try another reader. Persistent mismatch on
   one reader + clean reads on another = the reader (or its USB path).
3. **Compare successive reads.** Stable and validating wins; flaky bytes
   that move between reads confirm marginal contacts.
4. **Check the pcscd debug log** (`pcscd --foreground --debug`) for
   framing/parity errors around the reset — independent confirmation
   from below your program.
5. **Fall back, don't force it:** attempt `SCardConnect` with default
   parameters (T=0) rather than the suspect TA1 speed values, or abort
   with a clear "unreadable card" error instead of guessing.

Contrast with the [§5.9 Case study: one card, two ATRs](#59-case-study-one-card-two-atrs) adapter case: checksum **valid** but content
nonsense means the card emitted confused bytes (bad power during reset),
not wire noise. Same seating ladder applies — but the diagnosis (card
side vs wire side) tells you reseating, not cable-swapping, is the fix.

### 7.4 APDU-level troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| `67 00` on a hand-built command | Lc/Le vs body mismatch (wrong case, [§3.1 Command anatomy: identify the case from the bytes](#31-command-anatomy-identify-the-case-from-the-bytes)) | Recount: byte[4] must equal len−5 (case 3) or len−6 (case 4) |
| `6C xx` treated as failure | It isn't one | Resend identical command with `Le = xx` |
| `61 xx` then stopping | Data left unread on a T=0 card | GET RESPONSE loop until `90 00` ([§3.4 Recovery: the three resend patterns](#34-recovery-the-three-resend-patterns)) |
| `6D 00` / `6E 00` | Unknown INS/CLA — wrong app selected, or wrong card family | `SELECT` the right application first; confirm the INS exists on this card |
| `6A 86` on a read | P1/P2 offset past end of file | Read back FCP/file size, stay inside it |
| `69 86` as the *first* answer | Nothing selected yet | `SELECT` before anything else |
| Status bytes look like data | Off-by-two framing | Last 2 of `recv[:n]` are always SW ([§3.2 Response framing: knowing it ended](#32-response-framing-knowing-it-ended)); re-split |
| Short `n`, no error, data cut | `recv` buffer too small | Grow the buffer (`MaxBufferSize` / `MaxBufferSizeExtended`), re-issue |
| Works on reader A, fails on B | PPS/speed outcome differs per path | Force T=0 defaults, compare ATRs field by field ([§7.2 Why can the "same card" show different ATRs?](#72-why-can-the-same-card-show-different-atrs)) |
| Timeout on a write, state unknown | May have executed ([§3.4 Recovery: the three resend patterns](#34-recovery-the-three-resend-patterns)) | Status *read* first; retry the write only if the read proves it didn't land |

---

## 8. T=0 vs T=1: the two transport protocols

Both carry the same APDUs ([§3 APDUs: structure, framing, validation, recovery](#3-apdus-structure-framing-validation-recovery)); they differ in *how bytes cross the wire*. One I/O contact, both ways, not at once:

```
T=0  (byte handshake)
  host --> CLA INS P1 P2
  card <-- ACK (INS or INS xor FF)
  host --> data byte, ...
  card <-- 0x60 wait, or SW1 SW2

T=1  (blocks)
  host --> I-block (APDU + checksum)
  card <-- R-block (got it / resend)
  card <-- I-block (data + SW)
```

`SCardTransmit` hides this. You still send one APDU and get data+SW.

### 8.1 T=0 — byte-oriented, the old default

T=0 moves one byte at a time, with the card sending **procedure bytes**
to steer the transfer:

- Send the header (`CLA INS P1 P2`). The card replies with a procedure
  byte, not with an extra command you type:
  - **ACK** — the card repeats `INS` or `INS xor 0xFF`. That means
    “send the next data byte” or “I will send the next byte”. On a
    trace you see `A4` or `5B` after a SELECT header, not the word ACK.
  - **NULL** (`0x60`) — still working, wait.
  - **SW1 SW2** — the transfer is done.
  The reader/IFD consumes these bytes. `SCardTransmit` still looks like
  one APDU in, one APDU+SW out; your Go code does not send ACK.
- Consequence: a command and its response are interleaved, and large
  responses don't fit the simple flow. Two status words exist for that:
  - `SW1 = 0x61, SW2 = xx` — "I have `xx` more bytes; ask for them with
    **GET RESPONSE**: `00 C0 00 00 xx`." Repeat until `90 00`.
  - `SW1 = 0x6C, SW2 = xx` — "wrong expected length; resend with
    `Le = xx`."
- T=0 is chatty (a byte-level handshake per byte). It is the default
  when the ATR has no TD byte, and many cards still offer it. It is
  not universal: a card that only advertises T=1 — especially in
  specific mode, which locks T=1 and skips PPS — never speaks T=0.
  Trust the TD chain and `activeProtocol` from `SCardConnect`; use the
  matching PCI. See the T=1-only skeleton in
  [§11 Other contact cards](#11-other-contact-cards).

### 8.2 T=1 — block-oriented

T=1 wraps everything in **blocks** with checksums (LRC/EDC), like tiny
network packets:

- **I-blocks** carry APDU chunks; long APDUs are **chained** across
  several I-blocks, and long responses come back the same way — no GET
  RESPONSE dance needed.
- **R-blocks** acknowledge ("got it, send next" / "resend that").
- **S-blocks** carry control (resync, abort, waiting-time extension).
- Errors are detected by checksum and retried at the link layer, so
  T=1 is both faster and more robust on noisy lines.

### 8.3 What this means for your code

- `SCardConnect(…, SCardProtocolAny, …)` negotiates; the winning protocol
  comes back in `activeProtocol`.
- Pass the matching descriptor to transmit: `pcscgo.SCardPCIT1()` or
  `pcscgo.SCardPCIT0()` (read-only pointers owned by the library —
  never write through them).
- **`pcscgo.SCardTransmit` does not unwrap `61xx`/`6Cxx` for you.**
  On T=0 you must issue GET RESPONSE yourself and loop until `90 00`;
  on T=1 the card/reader handles chaining below your APDU.
- If `activeProtocol == SCardProtocolT0` in the README example, that is
  why it picks `SCardPCIT0()`.
- Extended-length APDUs (data > 255 bytes) need T=1 or T=0-with-extended
  support plus large buffers (`pcscgo.MaxBufferSizeExtended`); short
  APDUs max out at `pcscgo.MaxBufferSize` (264).

---

## 9. Putting it together: the monitoring loop

The canonical shape (see `examples/scan` for the complete version):

```go
// 1. Context + initial reader list (query size, then fill).
// 2. One SCardReaderState per reader (CurrentState: SCardStateUnaware)
//    + one "\\?PnP?\Notification" entry.
// 3. Loop:
//      SCardGetStatusChange(ctx, pcscgo.Infinite, states)
//      for each entry:
//          - mask out Changed + event counter; skip if nothing changed
//          - PnP entry changed → SCardListReaders again, diff, rebuild
//          - Present+Mute / ATRLength==0 → electrical problem (§7)
//          - Empty→Present → new card: parse ATR (§5), SCardConnect…
//      copy EventState → CurrentState for the next round
// 4. Ctrl+C → SCardCancel(ctx) from the signal handler unblocks the wait.
```

`SCardCancel` is the graceful-shutdown primitive: it turns the blocked
`SCardGetStatusChange` into `SCardErrorCancelled`, which your loop treats
as "time to exit".

---

## 10. Access control: voltage, write permission, PINs

### 10.1 Voltage is the reader's job, not yours

Your program **never sets a voltage** — there is no voltage parameter in
any PC/SC call. During activation the reader applies ground, then VCC,
then the clock, then releases RST ([§4 Power-up: reset, ATR, and PPS](#4-power-up-reset-atr-and-pps)), choosing among the card classes
(5 V, 3 V, 1.8 V) by negotiating with the card; pcscd and the reader
driver handle all of it. Voltage is therefore a hardware matter — but it
leaks into application code in two observable ways:

- `SCardStateUnpowered`: the card is present but powered down. Fix in
  software: `SCardConnect`/`SCardReconnect` powers it back up.
- `Present` + `Mute`, or `SCardConnect` failing on a card that works in
  another reader: the reader/card pair can't complete activation
  (marginal contacts, unsupported class, brownout through an adapter).
  Fix in hardware: reseat, clean, drop the adapter, try another reader.

If you see the first, reconnect. If you see the second, stop debugging
your code — it's the physics.

### 10.2 "Can I write to this file?"

There is **no PC/SC capability flag** for write access. Permission lives
inside the card, per file, as *security attributes*: every file (MF, DF,
EF — an *Elementary File* such as `EF_XXX`, the leaf
that actually holds bytes) declares which operations need which PIN or key. The
practical flow:

1. `SELECT` the file. The card returns an FCP (file control parameters)
   template describing the file — including its security attributes.
2. If the operation you want is PIN-gated, `VERIFY` the PIN first ([§10.3 PINs: when, which, how](#103-pins-when-which-how)).
3. Attempt the write (`UPDATE BINARY` / `UPDATE RECORD`). The card is the
   final arbiter — believe its status word:

| SW | Meaning | Action |
|----|---------|--------|
| `90 00` | Done | — |
| `69 82` | Security status not satisfied — PIN (or key) needed | `VERIFY`, then retry |
| `63 Cx` | Wrong PIN, X tries left | Retry with the right PIN; X = 0 means look at `69 83` |
| `69 83` | Authentication blocked — PIN dead | Unblock with PUK ([§10.3 PINs: when, which, how](#103-pins-when-which-how)) |
| `6A 82` | File not found | Wrong ID/path — not a permission problem |
| `69 86` | No current EF selected | `SELECT` first |

Typical SIM layout (confirm against GSM 11.11 / 3GPP 31.102 for the card
in use): some files read freely, some need PIN1 (e.g. phonebook, SMS),
some are updatable after `VERIFY`, and key files are never readable at
all. "Can I write EF_X?" is therefore answered empirically — SELECT,
check the FCP, VERIFY if gated, try — not by asking the reader.

### 10.3 PINs: when, which, how

**When is a PIN needed?** When a protected operation answers `69 82`.
There is no separate "PIN required?" query at the PC/SC level — the card
tells you by refusing. (Design your code PIN-first for files you know are
gated, rather than waiting for the refusal.)

**How do you give it?** With `VERIFY`: header `00 20 00 <P2>`, data = the
PIN. `P2` selects *which* PIN: on SIM/UICC cards `P2=01` is PIN1 and
`P2=02` is PIN2 (card-family-specific — check the card's spec).

**How many tries are left?** On UICC/SIM cards, send `VERIFY` with the
PIN reference but *no PIN data*: the card answers `63 Cx` with the
remaining counter **without consuming an attempt**. (UICC-specific
behavior — don't assume it on EMV or ID cards.)

**PIN dead (`69 83`)?** Unblock with `UNBLOCK` (`00 2C 00 <P2>`, data =
PUK + new PIN). Out of PUK tries = the card is permanently locked for
that PIN. That is a card-issuer problem, not a software bug.

**One correction to "PIN means encrypted":** a PIN gates *access*, not
*confidentiality*. File bytes on the card are often stored plainly; the
chip simply refuses to serve them until `VERIFY` succeeds. Real
encryption on the wire (secure messaging) is a separate, opt-in layer.
And PIN verification is **session-scoped**: after disconnect/reconnect
(or power loss) you verify again.

**Refused first, then VERIFY, then retry? Yes.** A command refused with
`69 82` was never executed — it has no side effects and consumes no PIN
tries (only failed `VERIFY` attempts decrement the `63 Cx` counter). The
recovery pattern is: send `VERIFY` with the PIN, expect `90 00`, then
re-issue the *identical* APDU. If `VERIFY` itself answers `69 83`, the
PIN is dead — switch to the PUK flow instead of retrying the command.

---

## 11. Other contact cards

`pcscgo` binds the whole PC/SC standard, so every contact card works the
same way up to `SCardTransmit` — only the ATR shapes and the APDU
dialects differ. Two constructed (but typical-shape — both machine-parsed
with [§5.7 Worked example (constructed, self-consistent)](#57-worked-example-constructed-self-consistent)'s procedure) examples to calibrate your eye:

**Minimal T=0-only card** (simple ID token style):
`3B 65 00 00 11 22 33 44 55`

T0=`65`: TB1 + TC1 present (`6`), K=5. No TD byte at all → **T=0 only**,
no TCK. Five opaque historical bytes, no TLV structure. This is the
simplest legal ATR shape: defaults for speed, one protocol, take-it-or-leave-it.

**Minimal T=1 card** (EMV-payment style):
`3B E0 00 00 11 01 F0`

T0=`E0`: TB1 + TC1 + TD1 present, K=0 — no historical bytes at all
(legal; identification happens at the application layer instead).
TD1=`11`: TA2 follows, protocol T=1. TA2=`01`: specific mode, T=1.
TCK=`F0` ✓. Real EMV ATRs are longer (more groups, historical bytes with
issuer data), but this skeleton — TD announcing T=1, TA2 locking it in —
is what you are looking for.

**The EMV dialect in one paragraph:** after `SCardConnect`, EMV apps
`SELECT` by AID (e.g. `A0 00 00 00 03 …` for Visa-type apps),
read the application's FCI (File Control Information) template, then pull data with `READ RECORD`
rather than `READ BINARY`. Records, PDOL/GPO negotiation, and
cryptograms are EMV-application specifics *above* the transport this
library provides — same wire, richer conversation. (Note the
`A0 00 00 …` shape here matches the application identifier found inside
the [§5.9 Case study: one card, two ATRs](#59-case-study-one-card-two-atrs) case study's historical TLV — the same numbering scheme shows
up in both places.)

---

## 12. Glossary

| Term | Meaning |
|------|---------|
| AID | Application Identifier — SELECT argument that names a DF |
| APDU | One command/response pair (CLA INS P1 P2 … + SW1 SW2) |
| ATR | Answer to Reset — card's self-description after reset ([§5 ATR anatomy, byte by byte](#5-atr-anatomy-byte-by-byte)) |
| CCID | Standard USB protocol most readers speak |
| Cold / warm ATR | After power-up / after reset-without-power-down |
| DF | Dedicated File — a directory; an application is usually a DF |
| EF | Elementary File — a leaf that holds bytes |
| etu | Elementary time unit — duration of one bit on the wire |
| FCI / FCP | File Control Information / File Control Parameters — what SELECT returns about the current file |
| Historical bytes | Identity blob at the tail of the ATR, before TCK ([§5.5 Historical bytes — the free-form ID card](#55-historical-bytes--the-free-form-id-card)) |
| IFD | Interface device (reader) + its driver |
| Lc / Le | Command data length / maximum response length wanted |
| MF | Master File — root of the card file tree |
| Mute | Present-but-silent card (`SCardStateMute`) |
| Multi-string | `\x00`-separated, `\x00\x00`-terminated name list |
| PCI | Protocol Control Info — T=0/T=1 descriptor for transmit |
| pcscd | Daemon owning the readers |
| PPS | Protocol and Parameters Selection (post-ATR negotiation) |
| SELECT | APDU INS `0xA4` — choose a DF or EF; not `SCardConnect` |
| SW1/SW2 | Two status bytes ending every response (`90 00` = OK) |
| T=15 | Global interface bytes in the ATR (class, clock stop) — not a transport protocol |

## 13. Where to go next

- Run `examples/scan -v` against a real reader and match its output to
  [§6 Detecting what happened: insert, remove, disconnect, mute](#6-detecting-what-happened-insert-remove-disconnect-mute)–[§7 Troubleshooting: the adapter-board scenario](#7-troubleshooting-the-adapter-board-scenario). It is the fastest way to make this concrete.
- Read the package godoc (`doc.go`) for exact function signatures.
- Card-type specifics live in the standards: ISO 7816-3 (electrical +
  ATR + T=0/T=1), ISO 7816-4 (APDUs, file system), GSM 11.11 / 3GPP
  31.102 (SIM/USIM files), EMV Books (payment).
- Primary sources: [PC/SC Workgroup](https://pcscworkgroup.com/),
  [pcsc-lite docs](https://pcsclite.apdu.fr/).

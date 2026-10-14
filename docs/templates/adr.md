# ADR Template

> Copy this file to `docs/adr/ADR-###-<slug>.md`

# ADR-###: <Title>

**Date:** YYYY-MM-DD
**Status:** Proposed | Accepted | Superseded | Deprecated
**Deciders:** 
**Related:** 

## Context
What is the issue that we’re trying to address?

## Decision Drivers
- ...

## Options Considered
1. **Option A**
   - Pros:
   - Cons:
2. **Option B**
   - Pros:
   - Cons:

## Decision Outcome
We choose **Option X** because...

## Consequences
### Positive
- ...

### Negative
- ...

## Compliance Notes
- Error handling contract: return up stack, no log-and-return, no empty catch
- Layer isolation per AGENTS.md

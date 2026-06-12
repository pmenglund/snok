# Snok Conformance Bundle

This bundle accompanies `../SPEC.md`. It translates Rune framework-core evidence into portable conformance cases for Snok, a Cobra-based framework layer.

Compatibility target: deterministic behavioral parity.

Source baseline: Rune framework core at revision `839ede33ce3644d3b60d5d5bd161a3217a425fda`.

## How To Use This Bundle

- Treat `../SPEC.md` as the normative behavior contract.
- Use `conformance.md` to identify required scenarios and comparison rules.
- Use `fixtures/*.json` as portable input/output cases for implementation tests.
- Use `evidence.md` to understand which Rune evidence informed the contract and which source assets were intentionally excluded.
- Use `manifest.json` to verify artifact integrity.

## Comparison Rules

- Fixture records are compared by structured JSON equality unless the case states another rule.
- Ordered arrays are order-sensitive.
- Object key order is not semantically significant.
- Text comparisons are semantic substring or section checks unless a fixture marks text as byte-exact.
- Paths in fixtures are logical paths, not host filesystem paths.

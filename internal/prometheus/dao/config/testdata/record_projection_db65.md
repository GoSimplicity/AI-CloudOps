# Record published payload fixture

`record_projection_db65.json` was captured by the actual SQL journal implementation at commit `db65def2ed37bca5956065d36a1d2de4bdf21134` in a clean local checkout, before validating the family3/4 shared implementation.

It covers empty and render plans, a live owner's retirement with deleted foreign history, accepted-artifact supersession, and legacy-owner supersession. Nonces, input content, IDs and historical deletion values are fixed; operation timestamps are outside these payloads. The completion calls establish SQL-only fixture facts and do not claim Redis or end-to-end acceptance.

The regression checks the published plan digest, manifest bytes and receipt representation. Fixture indentation is compacted for receipt comparison; property order, encodings and values are retained. Do not regenerate expected data using the implementation under review. The capture flag writes only to a new private file and is intended for explicitly recording a separately checked historical implementation.

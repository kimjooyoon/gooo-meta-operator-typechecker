# Falsifiable semantic boundary

The system under test is a closed miniature compiler fixture, not an
open-ended self-improving agent. The fixture accepts two declarations and
produces `module:<name>;term:<name>`. A generated proof can invoke only the
operation steps declared by the authoritative `.gooo` source.

The boundary is falsifiable through seven exact observations:

1. Two identity values compose with an empty effect row.
2. A fixture read and artifact normalization compose only when their effect
   rows are disjoint.
3. A `$cap` operation can be instantiated with a declared, bound capability.
4. A run-stage operation cannot escape into quote stage.
5. A generated-origin operation cannot capture caller origin.
6. A missing capability is `UNKNOWN`, with all six required fields.
7. Replay is `CLOSED` only when the second artifact digest equals the first
   and the replay effect trace matches the source declaration.

This does not prove termination, noninterference, capability safety outside
the declared rows, or correctness of an arbitrary compiler. It only proves
that this source-defined IR and its generated executor agree on the fixed
fixture and vector. Changing the source, contract identity, generated proof,
fixture output, terminal reason, or effect trace must either fail verification
or produce a non-`CLOSED` result.

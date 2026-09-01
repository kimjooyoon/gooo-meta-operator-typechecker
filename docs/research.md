# Research boundary and primary references

The design is intentionally limited to ideas that can be stated as concrete
checks in the `.gooo` source.

* MetaML provides the staged-computation model behind explicit stage indices:
  [MetaML: An Introduction to Multi-Stage Programming](https://www.cs.cmu.edu/~crary/819-f09/Flatt97.pdf).
* Davies and Pfenning formalize staged computation with modal types and show
  why code-level stage distinctions matter:
  [A Modal Analysis of Staged Computation](https://www.cs.cmu.edu/~fp/papers/jacm00.pdf).
* Flatt, Findler, and Felleisen describe hygienic macro expansion and lexical
  context preservation:
  [Scheme and Macros](https://www.cs.utah.edu/plt/publications/macromod.pdf).
* Koka's language paper is the primary reference for row-polymorphic effect
  tracking used as inspiration for effect rows:
  [Koka: Programming with Row-Polymorphic Effect Types](https://www.microsoft.com/en-us/research/wp-content/uploads/2016/02/koka-icfp2013.pdf).

These references motivate the vocabulary; they do not establish this
repository's results. The `.gooo` declarations are the only authority for
the fixed rules and expected terminal observations. The implementation makes
the narrower claim described in `semantic-boundary.md`, so a change to an
effect row, stage index, origin, capability binding, terminal reason, or
replay trace is testable as a changed proof or a failed verification.

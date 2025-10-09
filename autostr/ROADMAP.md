// TODO(later): Detect cycles that are reachable only via interface references.
// Idea: when v.Kind()==Interface, record identity of the underlying object
// (e.g., pointer address for pointer-concrete values, or a stable handle for
// non-pointer concretes) in a separate visitedInterface set to prevent loops
// in graph-like structures that hide behind interfaces.
// TODO: implement cache to reduce refleciton costs

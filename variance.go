package miruken

type Variance uint

const(
	Covariant Variance = iota
	Contravariant
	Invariant
)

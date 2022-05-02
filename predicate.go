package miruken

// Predicate represents a generic selector.
type Predicate[T any] func(T) bool

func combinePredicate2[T any](
	predicate1, predicate2 Predicate[T],
) Predicate[T] {
	if predicate1 == nil {
		return predicate2
	} else if predicate2 == nil {
		return predicate1
	}
	return func(val T) bool {
		if predicate2(val) {
			return true
		}
		if predicate1(val) {
			return true
		}
		return false
	}
}

func CombinePredicates[T any](
	predicate Predicate[T],
	predicates ... Predicate[T],
) Predicate[T] {
	switch len(predicates) {
	case 0: return predicate
	case 1: return combinePredicate2(predicate, predicates[0])
	default:
		for _, p := range predicates {
			predicate = combinePredicate2(predicate, p)
		}
		return predicate
	}
}

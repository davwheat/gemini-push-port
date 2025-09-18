package generic

func Map[I any, O any](xs []I, f func(I) O) []O {
	ys := make([]O, len(xs))
	for i, x := range xs {
		ys[i] = f(x)
	}
	return ys
}

package main

import "golang.org/x/exp/constraints"

func max[T constraints.Ordered](x, y T) T {
	if x < y {
		return y
	}
	return x
}

func min[T constraints.Ordered](x, y T) T {
	if x < y {
		return x
	}
	return y
}

func clamp[T constraints.Ordered](x, low, high T) T {
	return min[T](max[T](x, low), high)
}

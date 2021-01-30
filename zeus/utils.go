package main

import "fmt"

func checkRange(start, end int) error {
	if end > 0 && start > end || start < 0 {
		return fmt.Errorf("invalid range [%d;%d[", start, end)
	}
	return nil
}

func clampRange(start, end, len int) (int, int, error) {
	if end > 0 {
		if end > len {
			return 0, 0, fmt.Errorf("unsufficient data size %d for [%d;%d[", len, start, end)
		}
		return start, end, nil
	}
	if start >= len {
		return 0, 0, fmt.Errorf("unsufficient data size %d for [%d;%d[", len, start, len)
	}
	return start, len, nil
}

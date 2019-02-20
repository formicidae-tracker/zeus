package main

import (
	"fmt"

	"git.tuleu.science/fort/libarke/src-go/arke"
)

type UndeclaredDeviceError struct {
	c arke.NodeClass
}

func (e UndeclaredDeviceError) Error() string {
	return fmt.Sprintf("Require a '%s' but none declared", Name(e.c))
}

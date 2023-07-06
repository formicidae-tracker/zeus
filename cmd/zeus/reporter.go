package main

type Reporter interface {
	Report(chan<- struct{})
}

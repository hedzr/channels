// Copyright © 2019 Hedzr Yeh.

package channels_test

import (
	"github.com/hedzr/pools/channels"
	"testing"
)

func TestMerge(t *testing.T) {
	in := gen(2, 3)

	// Distribute the sq work across two goroutines that both read from in.
	c1 := sqi(in)
	c2 := sqi(in)

	// Consume the merged output from c1 and c2.
	for n := range channels.Merge(c1, c2) {
		t.Log(n) // 4 then 9, or 9 then 4
	}

	x := 1
	t.Log(channels.ZM(x))
}

func TestFanOut(t *testing.T) {
	// pipeline at first
	for n := range sqaure(sqaure(genseq(3))) {
		t.Log(n) // 1 then 16 then 81
	}

	// pipeline at first
	for n := range sqaure(genseq(13)) {
		t.Log(n) // 1 to 169
	}

	in := gen(2, 3)

	c1 := sqi(in)
	c2 := sqi(in)

	channels.FanOut(in, c1, c2)
}

func gen(nums ...channels.In) <-chan channels.In {
	out := make(chan channels.In)
	go func() {
		for _, n := range nums {
			out <- n
		}
		close(out)
	}()
	return out
}

func sqi(in <-chan channels.In) chan channels.Out {
	out := make(chan channels.Out)
	go func() {
		for n := range in {
			if ni, ok := n.(int); ok {
				out <- ni * ni
			}
		}
		close(out)
	}()
	return out
}

func genseq(upbound int) <-chan int {
	out := make(chan int)
	go func() {
		for i := 1; i <= upbound; i++ {
			out <- i
		}
		close(out)
	}()
	return out
}

func sqaure(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		for n := range in {
			out <- n * n
		}
		close(out)
	}()
	return out
}

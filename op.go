// Copyright © 2019 Hedzr Yeh.

package channels

import (
	"reflect"
	"sync"
)

// Data generic interface{}
type Data interface {
}

// In generic interface{}
type In interface {
}

// Out generic interface{}
type Out interface {
}

// TI transforms In to Out
func TI(in In) (out Out) {
	out = in
	return
}

// TD transforms Data to Out
func TD(in Data) (out Out) {
	out = in
	return
}

// Map map 'in' type channel to 'out' type channel
func Map(in <-chan In, transformer func(input In) (output Out)) (out <-chan Out) {
	o := make(chan Out)
	if in == nil {
		close(o)
	} else {
		out = o
		go func() {
			defer close(o)
			for v := range in {
				o <- transformer(v)
			}
		}()
	}
	return
}

// Reduce produces input continuously
func Reduce(in <-chan In, fn func(r Out, v In) Out) (out Out) {
	if in == nil {
		return nil
	}
	out = <-in
	for v := range in {
		out = fn(out, v)
	}
	return out
}

// OrDone selects input from channel or checking done signal triggering
func OrDone(done <-chan struct{}, c <-chan Data) <-chan Data {
	valStream := make(chan Data)
	go func() {
		defer close(valStream)
		for {
			select {
			case <-done:
				return
			case v, ok := <-c:
				if ok == false {
					return
				}
				select {
				case valStream <- v:
				case <-done:
				}
			}
		}
	}()
	return valStream
}

// Flat collects inputs and flatten to output
func Flat(done <-chan struct{}, chanStream <-chan <-chan Data) <-chan Data {
	valStream := make(chan Data)
	go func() {
		defer close(valStream)
		for {
			var stream <-chan Data
			select {
			case maybeStream, ok := <-chanStream:
				if ok == false {
					return
				}
				stream = maybeStream
			case <-done:
				return
			}
			for val := range OrDone(done, stream) {
				select {
				case valStream <- val:
				case <-done:
				}
			}
		}
	}()
	return valStream
}

// SkipN skips the first N elements
func SkipN(done <-chan struct{}, valueStream <-chan Data, num int) <-chan Data {
	takeStream := make(chan Data)
	go func() {
		defer close(takeStream)
		for i := 0; i < num; i++ {
			select {
			case <-done:
				return
			case <-valueStream:
			}
		}
		for {
			select {
			case <-done:
				return
			case takeStream <- <-valueStream:
			}
		}
	}()
	return takeStream
}

// SkipFn skips the elements in the stream while fn matched
func SkipFn(done <-chan struct{}, valueStream <-chan Data, fn func(Data) bool) <-chan Data {
	takeStream := make(chan Data)
	go func() {
		defer close(takeStream)
		for {
			select {
			case <-done:
				return
			case v := <-valueStream:
				if !fn(v) {
					takeStream <- v
				}
			}
		}
	}()
	return takeStream
}

// SkipWhile skips the elements while 'fn' met, and resume the stream once 'fn' not matched
func SkipWhile(done <-chan struct{}, valueStream <-chan Data, fn func(Data) bool) <-chan Data {
	takeStream := make(chan Data)
	go func() {
		defer close(takeStream)
		take := false
		for {
			select {
			case <-done:
				return
			case v := <-valueStream:
				if !take {
					take = !fn(v)
					if !take {
						continue
					}
				}
				takeStream <- v
			}
		}
	}()
	return takeStream
}

// TakeN return the first n elements from the valueStream
func TakeN(done <-chan struct{}, valueStream <-chan Data, num int) <-chan Data {
	takeStream := make(chan Data)
	go func() {
		defer close(takeStream)
		for i := 0; i < num; i++ {
			select {
			case <-done:
				return
			case takeStream <- <-valueStream:
			}
		}
	}()
	return takeStream
}

// TakeFn just returns the data while 'fn' matched ok
func TakeFn(done <-chan struct{}, valueStream <-chan Data, fn func(Data) bool) <-chan Data {
	takeStream := make(chan Data)
	go func() {
		defer close(takeStream)
		for {
			select {
			case <-done:
				return
			case v := <-valueStream:
				if fn(v) {
					takeStream <- v
				}
			}
		}
	}()
	return takeStream
}

// TakeWhile return those first data stream while 'fn' met, and cut-off once 'fn' failed
func TakeWhile(done <-chan struct{}, valueStream <-chan Data, fn func(Data) bool) <-chan Data {
	takeStream := make(chan Data)
	go func() {
		defer close(takeStream)
		for {
			select {
			case <-done:
				return
			case v := <-valueStream:
				if !fn(v) {
					return
				}
				takeStream <- v
			}
		}
	}()
	return takeStream
}

// Tee fan-out one input to each receivers
func Tee(in <-chan In, out ...chan Out) {
	go func() {
		defer func() {
			for i := 0; i < len(out); i++ {
				close(out[i])
			}
		}()
		for v := range in {
			for i := 0; i < len(out); i++ {
				out[i] <- v
			}
		}
	}()
}

// TeeAsync fan-out one input to each receivers asynchronously
func TeeAsync(in <-chan In, out ...chan Out) {
	go func() {
		defer func() {
			for i := 0; i < len(out); i++ {
				close(out[i])
			}
		}()
		for v := range in {
			for i := 0; i < len(out); i++ {
				go func() {
					out[i] <- v
				}()
			}
		}
	}()
}

// TeeReflect likes Tee but via golang reflect
func TeeReflect(in <-chan In, out ...chan Out) {
	go func() {
		defer func() {
			for i := 0; i < len(out); i++ {
				close(out[i])
			}
		}()
		cases := make([]reflect.SelectCase, len(out))
		for i := range cases {
			cases[i].Dir = reflect.SelectSend
		}
		for v := range in {
			v := v
			for i := range cases {
				cases[i].Chan = reflect.ValueOf(out[i])
				cases[i].Send = reflect.ValueOf(v)
			}
			for _ = range cases { // for each channel
				chosen, _, _ := reflect.Select(cases)
				cases[chosen].Chan = reflect.ValueOf(nil)
			}
		}
	}()
}

// Merge is a fan-out operator with random algorithm (based golang runtime)
func Merge(cs ...<-chan Out) <-chan Out {
	var wg sync.WaitGroup
	out := make(chan Out)

	// Start an output goroutine for each input channel in cs.  output
	// copies values from c to out until c is closed, then calls wg.Done.
	output := func(c <-chan Out) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}
	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	// Start a goroutine to close out once all the output goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

// FanOut with round-robin balancing algorithm.
func FanOut(in <-chan In, out ...chan Out) {
	go func() {
		defer func() {
			for _, c := range out {
				close(c)
			}
		}()

		i, n := 0, len(out)
		for v := range in {
			out[i] <- v
			i = (i + 1) % n
		}
	}()
}

// FanOutReflect using reflect
func FanOutReflect(ch <-chan In, out ...chan Out) {
	go func() {
		defer func() {
			for _, c := range out {
				close(c)
			}
		}()

		cases := make([]reflect.SelectCase, len(out))
		for i := range cases {
			cases[i].Dir = reflect.SelectSend
			cases[i].Chan = reflect.ValueOf(out[i])
		}

		for v := range ch {
			for i := range cases {
				cases[i].Send = reflect.ValueOf(v)
			}
			_, _, _ = reflect.Select(cases)
		}
	}()
}

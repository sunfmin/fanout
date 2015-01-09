package fanout

import (
	"errors"
	// "fmt"
	"sync"
)

// feedInputs starts a goroutine to loop through inputs and send the
// input on the interface{} channel. If done is closed, feedInputs abandons its work.
func feedInputs(done <-chan int, inputs []interface{}) (<-chan interface{}, <-chan error) {
	inputsChan := make(chan interface{})
	errChan := make(chan error, 1)

	go func() {
		// Close the inputs channel after feedInputs returns.
		defer close(inputsChan)

		// No select needed for this send, since errc is buffered.
		errChan <- func() error {
			for _, input := range inputs {
				select {
				case inputsChan <- input:
				case <-done:
					// fmt.Println("feedInput Done")
					return errors.New("loop canceled")
				}
			}
			return nil
		}()
	}()
	return inputsChan, errChan
}

type resultWithError struct {
	result interface{}
	err    error
}

func work(done <-chan int, inputs <-chan interface{}, c chan<- resultWithError, w Worker) {
	for input := range inputs {
		// fmt.Println("Got ", input, " in worker")
		re := resultWithError{}
		re.result, re.err = w(input)
		select {
		case c <- re:
		case <-done:
			// fmt.Println("worker done.")
			return
		}
	}
}

// If you have multiple params, You can wrap them into one struct,
// For multiple result, You can wrap them into on result struct,
// In your work, If you return any error, The whole parallel run thing will stop immediately.
//  If you want to ignore Error in some of the workers, Then return nil error in your Worker func.
type Worker func(input interface{}) (result interface{}, err error)

func ParallelRun(workerNum int, w Worker, inputs []interface{}) (results []interface{}, err error) {
	// closes the done channel when it returns; it may do so before
	// receiving all the values from c and errc.
	done := make(chan int)
	defer close(done)

	inputsc, errc := feedInputs(done, inputs)
	// fmt.Printf("errc = %#v, %d\n", errc, len(errc))

	// Start a fixed number of goroutines to do the worker.
	c := make(chan resultWithError)
	var wg sync.WaitGroup
	wg.Add(workerNum)

	for i := 0; i < workerNum; i++ {
		// fmt.Println("starting ", i)
		go func() {
			work(done, inputsc, c, w)
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(c)
	}()

	for r := range c {
		if r.err != nil {
			return nil, r.err
		}
		results = append(results, r.result)
	}

	// Check whether the feedInputs failed.
	if err := <-errc; err != nil {
		return nil, err
	}

	return
}

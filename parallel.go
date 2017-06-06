/*Package fanout package provide a simple way for people who are difficult to write concurrency program that use
channels + goroutines + WaitGroup combination. I often find it difficult to write correctly.
more documentation and example at: https://github.com/sunfmin/fanout
*/
package fanout

import (
	"errors"
	"sync"
)

// feedInputs starts a goroutine to loop through inputs and send the
// input on the interface{} channel. If done is closed, feedInputs abandons its work.
func feedInputs(done <-chan int, inputs IArray) (<-chan interface{}, <-chan error) {
	inputsChan := make(chan interface{})
	errChan := make(chan error, 1)

	go func() {
		// Close the inputs channel after feedInputs returns.
		defer close(inputsChan)

		// No select needed for this send, since errc is buffered.
		errChan <- func() error {
			l := inputs.Len()
			for i := 0; i < l; i++ {
				select {
				case inputsChan <- inputs.Get(i):
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

type IArray interface {
	Get(int) interface{}
	Len() int
}

// Worker is the interface to be implemented when using this helper package
// If the Worker func needs to have multiple params, You can wrap them into one struct,
// Also for multiple result, You can wrap them into one result struct,
// In Worker, If it return any error, All the other workers will stop immediately.
// If you want to ignore Error in some of the workers, Then return nil error in your Worker func.
type Worker func(input interface{}) (interface{}, error)

// ParallelRun starts `workerNum` of goroutines immediately to consume the value of inputs, and provide input to `Worker` func.
// and run the `Worker`, If any worker finish, it will put the result value into a channel, then append to the results value.
// The func will block the execution and wait for all goroutines to finish, then return results all together.
func ParallelRun(workerNum int, w Worker, inputs IArray) ([]interface{}, error) {
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

	results := []interface{}{}
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

	return results, nil
}

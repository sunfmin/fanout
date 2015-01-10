## Fanout - make writing parallel code even easier

This code is port from sample code of Go blog post [Go Concurrency Patterns: Pipelines and cancellation](http://blog.golang.org/pipelines)'s Bounded parallelism section [sample code](http://blog.golang.org/pipelines/bounded.go)

I made the fanout pattern sample code reusable, So you can easily write parallel code without worry about `fatal error: all goroutines are asleep - deadlock!`, Which I encountered quite often when trying to write parallel code, and difficult to figure out why.

From the blog post:

> Multiple functions can read from the same channel until that channel is closed; this is called fan-out. This provides a way to distribute work amongst a group of workers to parallelize CPU use and I/O.


For a big list of data, You want to go though each of them and do something with them in parallel. and get the result for each of them to another list for later use. But normally you don't start a new goroutine for every one of them, Because you might don’t know how big is the list, and if it’s too big, that would eat too much memory. This package made write this kind of program easier.


## API

```go
type Worker func(input interface{}) (result interface{}, err error)
func ParallelRun(workerNum int, w Worker, inputs []interface{}) (results []interface{}, err error) {
```

The package contains these two public api:

1. `Worker` is the custom code you want to run,
2. `ParallelRun` start to run the `Worker`
   - `workerNum`: start how many goroutines to consume the inputs list, it could be larger than your list size, or smaller, If it’s larger, some of the goroutines will run empty because can’t get input to work from the channel.
   - `inputs`: the inputs list that you first need to convert your list to `[]interface{}`
   - `results`: the result list that returned from the `Worker`, you normally want to go through them and cast them back to your the real type the `Worker` actually returns.


## A chart to describe how it works

After you fire up the `ParallelRun` func, It will instantly start the `workerNum` goroutines, and start to simultaneously work on the Inputs List. After all workers finished without error, It will return the results list. If any of the workers return error. All the other workers will immediately stop and return the first error to `ParallelRun`


```

                                   (goroutines)
ParallelRun method:            +------------------+
                             ++|     worker 1     |++
                             | +------------------+ |
                             | |     worker 2     | |
                             | +------------------+ |
                             | |     worker 3     | |
     Inputs List             | +------------------+ |    Output List (random order)
+------------------------+   | |     worker 4     | |    +----+----+----+----+----+
| 1  | 2  | 3  | 4  | 5  | +-+ +------------------+ +--> | o1 | o2 | o3 | o4 | o5 |
+----+----+----+----+----+   | |     worker 5     | |    +----+----+----+----+----+
                             | +------------------+ |
                             | |     worker 6     | |
                             | +------------------+ |
                             | |     worker 7     | |
                             | +------------------+ |
                             | |     worker 8     | |
                             | +------------------+ |
            workerNum: 9     ++|     worker 9     |++
                               +------------------+

```



## Example: check my ideal domain is available or not

For example, I have a text file contains Chinese words like this

```
小米
翻译
老兵
电影
土豆
豆瓣
文章
圆通
做爱
迅雷
昆明
地图
... and thousands more

```


I want to do first to convert the word to PinYin, and then suffix `.com`, Then use `whois` command to all of them to see if they are still available. So that if they are, I can quickly register it.

The non-parallel slow program will look like this:

```go

for _, word := range domainWords {
	if strings.TrimSpace(word) == "" {
		continue
	}

	py := pinyin.Convert(word)
	pydowncase := strings.ToLower(py)
	domain := pydowncase + ".com"
	outr, err := domainAvailable(word, domain)
	if err != nil {
		fmt.Println("Error: ", err)
		continue
	}

	if outr.available {
		fmt.Printf("[Ohh Yeah] %s %s\n", outr.word, outr.domain)
		continue
    }
	
	fmt.Printf("\t\t\t %s %s %s\n", outr.word, outr.domain, outr.summary)
}

type checkResult struct {
	word      string
	domain    string
	available bool
	summary   string
}

func domainAvailable(word string, domain string) (ch checkResult, err error) {
	var summary string
	var output []byte

	ch.word = word
	ch.domain = domain

	cmd := exec.Command("whois", domain)
	output, err = cmd.Output()
	if err != nil {
		fmt.Println(err)
		return
	}

	outputstring := string(output)
	if strings.Contains(outputstring, "No match for \"") {
		ch.available = true
		return
	}

	summary = firstLineOf(outputstring, "Registrant Name") + " => "
	summary = summary + firstLineOf(outputstring, "Expiration Date")
	ch.summary = summary
	return
}

```

This would be ridiculous slow, Because for each word, the program start to run command `whois`, and after it finish, It start to run next word. It would take days to run for example 30000 words.

## Change it to parallel with fanout

You need to install the package if you haven't:

```
go get github.com/sunfmin/fanout
```

And with one call, you get it all parallel:

```go
inputs := []interface{}{}


for _, word:= range domainWords {
	inputs = append(inputs, word)
}

results, err2 := fanout.ParallelRun(60, func(input interface{}) (interface{}, error) {
	word := input.(string)
	if strings.TrimSpace(word) == "" {
		return nil, nil
	}

	py := pinyin.Convert(word)
	pydowncase := strings.ToLower(py)
	domain := pydowncase + ".com"
	outr, err := domainAvailable(word, domain)
	if err != nil {
		fmt.Println("Error: ", err)
		return nil, err
	}

	if outr.available {
		fmt.Printf("[Ohh Yeah] %s %s\n", outr.word, outr.domain)
		return outr, nil
	}
	fmt.Printf("\t\t\t %s %s %s\n", outr.word, outr.domain, outr.summary)
	return outr, nil
}, inputs)

fmt.Println("Finished ", len(results), ", Error:", err2)

```

The full runnable code is here: https://github.com/sunfmin/domaincheck/blob/master/main.go

## Q&A

1. Does `fanout.ParallelRun` block execution?

   Yes, it does, It will wait until all the goroutines it started to finish, then collect all the results into the returned []interface{} value, and you might want to unwrap it and sort it for later use.

2. How does it make the program run faster?

   For example there is a list contains 20 elements need to process, if the func for processing one elements takes exactly 1 second. In a non-parallel way, It basically will spend 20 seconds to do the work and show you the result of 20 elements. But by using `fanout.ParallelRun`, if you set the `workerNum` to be 20, It totally will only spend the longest execution time 1 second to finish the total 20 elements. So it's a 20x improvement. In reality it won't be exactly 20x improvement. But it maximize CPU usage and I/O throughput.


Enjoy!


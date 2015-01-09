## Fanout - make writing parallel code easier

This code is port from sample code of Go blog post [Go Concurrency Patterns: Pipelines and cancellation](http://blog.golang.org/pipelines)'s Bounded parallelism section [sample code](http://blog.golang.org/pipelines/bounded.go)

I made the fanout pattern code reusable, So you can easily write parallel code without worry about `fatal error: all goroutines are asleep - deadlock!`, Which I encoutered quite often when trying to write parallel code, and difficult to figure out why.

Say you have a big list of data, You want to go though each of them and do something with them. And get the result for each of them to another list.


## API

```
type Worker func(input interface{}) (result interface{}, err error)
func ParallelRun(workerNum int, w Worker, inputs []interface{}) (results []interface{}, err error) {
```

The package contains these two public api:

1. `Worker` is the custom code you want to run,
2. `ParallelRun` start to run the `Worker`
   - `workerNum`: start how many goroutines to comsume the inputs list
   - `inputs`: the inputs list that you first need to convert your list to `[]interface{}`
   - `results`: the result list that returned from the `Worker`


## Example: Check my ideal domain is available or not

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


I want to do first to convert the word to PinYin, and then suffix `.com`, Then use `whois` to all of them to see if they are still available. So that if they are, I can quickly register it.

The non parallel program will look like this:

```

for _, word := range domainWords {
	py := pinyin.Convert(word)
	pydowncase := strings.ToLower(py)
	domain := pydowncase + ".com"

	ch, err := domainAvailable(word, domain)
	println(ch)
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

```
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

	if outr.available {
		fmt.Printf("[Ohh Yeah] %s %s\n", outr.word, outr.domain)
	} else {
		fmt.Printf("\t\t\t %s %s %s\n", outr.word, outr.domain, outr.summary)
	}

	if err != nil {
		fmt.Println("Error: ", err)
	}

	return outr, nil
}, inputs)

fmt.Println("Finished ", len(results), ", Error:", err2)

```

The full runnable code is here: https://github.com/sunfmin/domaincheck/blob/master/main.go

Enjoy!


#####  Dear sunfmin，
#####  我是【溜达】的HR，我们正在为寻找优秀的GO语言开发者！
#####  我在GitHub上看到您的作品，真诚的对您发出邀请，希望可以和您取得联系，获得一次和您交流的机会！
#####  同时已向您的邮箱（sunfmin@gmail.com），发送了一封邀请函，希望能够得到您的回复！


##### 以下是公司相关介绍：
[![](https://static.lagou.com/thumbnail_300x300/i/image/M00/6A/19/CgpEMlmmlYSAQRDRAABAkrB_kQ8059.png)](https://www.52liuda.com/)
*【溜达】—从这里开始，探索世界*

我们相信在我们生活的物理空间内，有很多有意思的事物或人。

我们相信太多有价值的信息就在我们身边但是都被我们错过。

我们相信【溜达】是能够帮助我们察觉生活中那些即美好又精彩的事物，或者遇见一生中绝对不想错过的人。

#### 【溜达】成立于2017年初，总部位于杭州，千万天使轮。在互联网领域已深耕十多年，在互联网营销和大数据方面有一整套独特的方法和工具。
#### 创始人【国平】，从事在线营销15年，带领团队以咨询顾问的角色，帮助携程、阿里云等近40家知名互联网公司获得了很大的市场份额。我们的团队大牛与小鲜肉兼具，乐于面对各种技术难题，相信没有什么问题是不能通过技术手段解决的，也坚信只要有好的idea，碰撞上足够的创新和勤奋就能实现它。

##### 我们提供高于行业的薪资和各种个性化福利待遇 ：
* 弹性工作时间、上下班无需打卡
* 社保五险、全额公积金、年终奖、绩效奖金、法定假日、假日礼物、带薪年假
* 行业大牛CEO的独家内部培训分享会，助攻UP~!
* 段位高级—超高能装备：升降支架，机械键盘，3D显示屏幕。
* 不定期组织outing 漂流，轰趴，露营BBQ，旅游等等超酷团建项目
* 免费购书无上限
* 核心员工股权激励方案




## Fanout - make writing parallel code even easier

This code is port from sample code of Go blog post [Go Concurrency Patterns: Pipelines and cancellation](http://blog.golang.org/pipelines)'s Bounded parallelism section [sample code](http://blog.golang.org/pipelines/bounded.go)

I made the fanout pattern sample code reusable, So you can easily write parallel code without worry about `fatal error: all goroutines are asleep - deadlock!`, Which I encountered quite often when trying to write parallel code, and difficult to figure out why.

From the blog post:

> Multiple functions can read from the same channel until that channel is closed; this is called fan-out. This provides a way to distribute work amongst a group of workers to parallelize CPU use and I/O.


For a big list of data, You want to go though each of them and do something with them in parallel. and get the result for each of them to another list for later use. But normally you don't start a new goroutine for every one of them, Because you might don’t know how big the list is, and if it’s too big, that would eat too much memory. This package makes writing this kind of program easier.


## API

```go
type Worker func(input interface{}) (result interface{}, err error)
func ParallelRun(workerNum int, w Worker, inputs []interface{}) (results []interface{}, err error) {
```

The package contains these two public api:

1. `Worker` is the custom code you want to run,
2. `ParallelRun` start to run the `Worker`
   - `workerNum`: how many goroutines to start consuming the inputs list, it could be larger than your list size, or smaller, If it’s larger, some of the goroutines will run empty because they can’t get input to work from the channel.
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

This would be ridiculously slow, Because for each word, the program start to run command `whois`, and after it finishes, It start to run next word. It would take days to run for example 30000 words.

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

   Yes, it does, It will wait until all the running goroutines finish, then collect all the results into the returned []interface{} value, and you might want to unwrap it and sort it for later use.

2. How does it make the program run faster?

   For example there is a list contains 20 elements need to process, if the func for processing one elements takes exactly 1 second. In a non-parallel way, It basically will spend 20 seconds to do the work and show you the result of 20 elements. But by using `fanout.ParallelRun`, if you set the `workerNum` to be 20, In total, it will only spend the longest execution time 1 second to finish the 20 elements. So it's a 20x improvement. In reality it won't be exactly a 20x improvement. Because of the cores of CPU, and how intensive the program is using I/O. But it will maximize CPU usage and I/O throughput.


Enjoy!


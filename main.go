package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

func HandleError(err error, why string) {
	if err != nil {
		fmt.Println(why, err)
	}
}

// 下载文件，传入url和文件名
func DownloadFile(url string, filename string) (ok bool) {
	resp, err := http.Get(url)
	HandleError(err, "http.get url")
	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	HandleError(err, "resp.body")
	filename = "F:/go爬虫/image/" + filename
	// 写出数据
	err = ioutil.WriteFile(filename, bytes, 0666)
	if err != nil {
		return false
	} else {
		return true
	}
}

// 并发爬取：
// 1，初始化数据管道
// 2，爬虫写出：26个协程向管道中添加图片链接
// 3，任务统计协程：检查26个任务是否都完成，完成则关闭数据管道
// 4，下载协程：从管道读取链接并下载

var (

	chanImageUrls chan string // 存放图片链接的数据管道
	chanTask chan string // 用于监控协程
	wg            sync.WaitGroup
	reImg    = `https?://[^"]+?(\.((jpg)|(png)|(jpeg)|(gif)|(bmp)))`
)

// 图片下载协程
// 从管道获取图片的具体链接，进行下载
func DownLoadImg() {
	for url := range chanImageUrls {
		filename := GetFilenameFromUrl(url)
		ok := DownloadFile(url, filename)
		if ok {
			fmt.Printf("%s 下载成功\n", filename)
		} else {
			fmt.Printf("%s 下载失败\n", filename)
		}
	}
	wg.Done()
}

// 从url截取文件名字
func GetFilenameFromUrl(url string) (filename string) {
	// 返回最后一个/的位置
	lastIndex := strings.LastIndex(url, "/")
	// 截取
	filename = url[lastIndex+1:]
	// 利用时间戳解决重名
	timePrefix := strconv.Itoa(int(time.Now().UnixNano()))
	filename = timePrefix + "_" + filename
	return
}

// 任务统计协程
// 从chanTask获取数据，若计数完毕，则关闭图片url管道，不再往里写入数据
func checkOK() {
	var count int
	for {
		<- chanTask
		fmt.Printf("%d 号完成了爬取任务\n", count)
		count ++
		if count == 26 {
			close(chanImageUrls)
			break
		}
	}
	wg.Done()
}

// 爬取图片链接到管道中
// url传的是整页的链接
func getImgUrls(url string) {
	imgUrls := getImgs(url)
	// 遍历切片里的所有链接，存入数据管道
	for _, imgUrl := range imgUrls {
		chanImageUrls <- imgUrl
	}
	// 标识当前协程完成
	// 每完成一个任务，写一个数据到监控管道中
	// 用于监控协程，知道已经完成了几个任务
	chanTask <- url
	wg.Done()
}

// 获取当前页面的图片链接
func getImgs(url string) (urls []string) {
	page := getPage(url)
	re := regexp.MustCompile(reImg)
	results := re.FindAllStringSubmatch(page, -1)
	fmt.Printf("共找到%d条结果\n", len(results))
	for _, result := range results {
		url := result[0]
		urls = append(urls, url)
	}
	return
}

// 获取页数
func getPage(url string) (page string) {
	resp, err := http.Get(url)
	HandleError(err, "http.Get url")
	defer resp.Body.Close()
	// 读取页面内容
	bytes, err := ioutil.ReadAll(resp.Body)
	HandleError(err, "ioutil.ReadAll resp.Body")
	// 字节转字符串
	page = string(bytes)
	return
}

func main() {

	startTime := time.Now().Unix()
	// 1.初始化管道
	chanImageUrls = make(chan string, 1000000)
	chanTask = make(chan string, 26)
	// 2.爬虫协程，此处26即开启26个协程，爬取26页的数据
	url := "https://www.bizhizu.cn/shouji/tag-%E5%8F%AF%E7%88%B1/"
	for i := 0; i < 26; i++ {
		wg.Add(1)
		go getImgUrls( url + strconv.Itoa(i) + ".html")
	}
	// 3.任务统计协程：统计26个任务是否都完成，完成则关闭管道
	wg.Add(1)
	go checkOK()
	// 4.下载协程：从管道中读取图片链接并下载
	// 此处设置协程数5，10，20，爬取的总耗时都为30s，原因？
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go DownLoadImg()
	}
	wg.Wait()
	endTime := time.Now().Unix()
	fmt.Printf("爬取完毕，总耗时%ds\n", endTime - startTime)
}

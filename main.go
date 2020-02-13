package main

import (
	"flag"
	"fmt"
	"sync"
)

var (
	maxGo     int = 15        //最大并发数
	imgSize   int = 1024 * 30 //图片大于30K,才下载
	MaxLayer  int = 3         //查找网页的最大层数,用于图片下载
	waitGroup sync.WaitGroup
	m3u8Url   string //m3u8下载的URL
	imgUrl    string //img下载的URL
)

func main() {

	flag.StringVar(&m3u8Url, "m", "", "M3U8 url index file")
	flag.StringVar(&imgUrl, "i", "", "img url")
	flag.IntVar(&maxGo, "c", 15, "maximum number of goroutine")
	flag.IntVar(&imgSize, "s", 30, "Only download the above pictures (KB)")
	flag.IntVar(&MaxLayer, "l", 3, "Download page Max Layer")
	flag.Parse()

	switch {
	case m3u8Url != "":
		err := dowloadM3u8(m3u8Url)
		if err != nil {
			fmt.Println(err)
			return
		}
	case imgUrl != "":
		imgSize = 1024 * imgSize
		DownloadImg(imgUrl)

	default:
		fmt.Println(`The parameter (m  "M3U8 URL") or (i "Img url") must be entered.`)
		return
	}

}

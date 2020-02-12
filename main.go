package main

import (
	"flag"
	"fmt"
	"sync"
)

var (
	maxGo     int = 15  //最大并发数
	waitGroup sync.WaitGroup
	m3u8Url   string  //m3u8下载的URL
	imgUrl    string  //img下载的URL
)

func main() {

	flag.StringVar(&m3u8Url, "m", "", "M3U8 url index file")
	flag.StringVar(&imgUrl, "i", "", "img url")
	flag.IntVar(&maxGo, "c", 15, "maximum number of goroutine")
	flag.Parse()

	switch {
	case m3u8Url != "":
		err := dowloadM3u8(m3u8Url)
		if err != nil {
			fmt.Println(err)
			return
		}
	case imgUrl != "":
		DownloadImg(imgUrl)

	default:
		fmt.Println(`The parameter (m or i) "M3U8 or Img url" must be entered.`)
		return
	}

}

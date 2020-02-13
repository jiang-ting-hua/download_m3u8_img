package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	//rePhone = `(1[3456789]\d)(\d{4})(\d{4})`
	//reEmail = `[\w\.]+@\w+\.[a-z]{2,3}(\.[a-z]{2,3})?`
	//relink  = `<a[\s\S]+?href="(http[\s\S]+?)"`                    //正则取链接.http开头的.
	//relink2 = `<a[\s\S]+?href="(read\.php[\s\S]{5}15[\s\S]{10}?)"` //正则取后面的的链接 <a href="read.php?tid-1532336.html"
	//relink3 = `<a[\s\S]+?href="(read\.php[\s\S]{17,27}?)"`         //正则取后面的的链接 <a href="read.php?tid-1508927-fpage-10.html"
	//reImg = `<img[\s\S]+?src=([\s\S]+)?>`
	//reImg          = `<img.+?src="(.+?)[\?"].*?>`
	//reImgAlt = `<img.+?alt="(.+?)"` //取img中的文件名
	//reAlt    = `alt="(.+?)"`        //取alt标签中的Alt属性 <img src="http://img.xfjw.net/templates/xfjw/img/logo.jpg" width="241" height="101"/>
	reImgName      = `/([\w-]+\.((jpg)|(jpeg)|(png)|(bmp)|(webp)|(swf)|(ico)))`
	reImgSuffix    = `\.((jpg)|(jpeg)|(png)|(gif)|(gif)|(bmp)|(webp)|(swf))`
	relinkAll      = `<a[\s\S]+?href="([\s\S]+?)"` //正则取链接<a href=开头包含的数据.
	reImg          = `<img([\s\S]+?)>`
	reCharacterSet = `<meta.+?charset=(.+?)"`
	TryImgNum      = 2 //下载失败,再次重试下载次数.
)

var (
	imgWG          sync.WaitGroup
	DowloadImgPath = "./img/"    //下载img图片保存路径.当前目录下的img目录.
	linkUrls       = []string{}  //需要下载的所有URL
	linkImgs       = sync.Map{}  //需要下载的图片信息
	Tryimg         = []imgInfo{} //保存失败的下载,提供再次尝试下载
	urlChan        chan string   //并发初始化所有url,提取要下载的图片链接
	imgChan        chan *imgInfo //并发下载图片文件的管道.
	imgFailed      chan *imgInfo //存放下载图片失败的管道
)

type imgInfo struct {
	imgUrl     string //图片文件的URL下载地址.
	fileName   string //图片文件名
	suffix     string //文件后缀
	isDownload bool   //是否下载成功
}

func init() {
	now := time.Now()
	formatNow := now.Format("20060102_15-04-05")
	DowloadImgPath = DowloadImgPath + formatNow + "/" //下载img图片保存路径.
}

//下载img
func DownloadImg(url string) {
	start := time.Now()
	//一.初始化需要下载的信息
	fmt.Println("Start initialization img url index. Please wait.")
	fmt.Println("Download only pictures larger than 30K.")
	//获得所有要下载的url网址
	getUrlAll(url)

	//初始化所有网址到管道
	initUrlChan()
	//并发下载所有网址的图片链接
	for i := 0; i < maxGo; i++ {
		imgWG.Add(1)
		go getUrlImgGo()
	}
	imgWG.Wait()
	//初始化管道
	initImgChan()

	//二.开始并发下载img文件,并发默认设为15,可以命令行调整.
	fmt.Println("Start dowload img file.")
	for i := 0; i < maxGo; i++ {
		waitGroup.Add(1)
		go DownloadImgGo()
	}
	waitGroup.Wait()

	//尝试重新下载失败的.
	tryFailedImg()

	cost := time.Since(start)
	fmt.Println("\nDownload completed.")
	fmt.Printf("Total download time =[%s]\n", cost)
}

//并发下载图片,保存在当前目录的./img目录下。
func DownloadImgGo() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("DownloadImg panic", err)
			waitGroup.Done()
		}
	}()
	for img := range imgChan {
		if img.isDownload == true {
			continue
		}
		start := time.Now()
		resp, err := getUrlResp(img.imgUrl)
		if err != nil {
			img.isDownload = false
			imgFailed <- img
			//fmt.Println("getUrlResp():", err)
			continue
		}
		defer resp.Body.Close()
		imgBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			img.isDownload = false
			imgFailed <- img
			//fmt.Println("ioutil.ReadAll():", err)
			continue
		}

		if len(imgBytes) < imgSize {
			img.isDownload = true
			continue
		}

		mkdir(DowloadImgPath)

		err = ioutil.WriteFile(img.fileName, imgBytes, 0644)
		if err != nil {
			img.isDownload = false
			imgFailed <- img
			//fmt.Println("ioutil.WriteFile():", err)
			continue
		}
		img.isDownload = true
		cost := time.Since(start)
		fmt.Printf("Download:%s Time:%s \n", img.fileName, cost)
	}
	waitGroup.Done()
}
func getUrlNext(url string, upper []string) (current []string) {
	if MaxLayer == 0 {
		return nil
	}
	if len(upper) == 0 {
		return nil
	}
	host := getHost(url)
	for _, v := range upper {
		fmt.Println("init url: ", v)
		link := getUrlLink(v)
		if len(link) == 0 {
			return nil
		}
		for _, v := range link {
			if v == url || v == host || v == host+`\` {
				continue
			}
			if isContains(linkUrls, v) == false {
				linkUrls = append(linkUrls, v)
				current = append(current, v)
			}
		}
	}
	MaxLayer--
	return getUrlNext(url, current)
}

//得到所要下载图片的页面.
func getUrlAll(url string) {
	//添加第一层链接
	linkUrls = append(linkUrls, url)
	//添加第二层的链接.
	//current2 := getUrlNext(linkUrls)
	getUrlNext(url, linkUrls)

	////添加第三层的链接
	//if len(current2) == 0 {
	//	return
	//}
	//getUrlNext(current2)
}
func initUrlChan() {
	urlChan = make(chan string, len(linkUrls))
	for _, v := range linkUrls {
		urlChan <- v
	}
	close(urlChan)
}

//根据正则取得的数据,进行筛选图片Url.
func buildImg(value [][]string) (imgs []string) {
	if len(value) == 0 {
		return
	}
	s1 := "data-original="
	s2 := "src="

	for i := 0; i < len(value); i++ {
		t := value[i][1]
		isUrl, _ := isUrlSuffix(t)
		if isUrl == true {
			continue
		}

		if strings.Contains(t, s1) {
			i := strings.Index(t, s1)
			t = t[i+len(s1):]
			t = cutUrl(t, `"`)
			t = strings.TrimSpace(t)
			if len(t) < 5 {
				continue
			}

			//判断是否有重复的URL
			if isContains(imgs, t) == false {
				imgs = append(imgs, t)
			}

		} else if strings.Contains(t, s2) {
			i := strings.Index(t, s2)
			t = t[i+len(s2):]
			t = cutUrl(t, `"`)
			t = strings.TrimSpace(t)
			if len(t) < 5 {
				continue
			}

			//判断是否有重复的URL
			if isContains(imgs, t) == false {
				imgs = append(imgs, t)
			}
		} else {
			continue
		}
	}
	return
}

//并发取得所有页面的图片链接
func getUrlImgGo() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("getUrlImgGo panic", err)
			imgWG.Done()
		}
	}()

	for url := range urlChan {
		Resp, err := getUrlResp(url)
		if err != nil {
			//err = fmt.Errorf("getUrl():", err)
			continue
		}
		defer Resp.Body.Close()
		bytes, err := ioutil.ReadAll(Resp.Body)
		if err != nil {
			//err = fmt.Errorf("ioutil.ReadAll():", err)
			continue
		}
		htmlText := string(bytes)

		//如果没有换行,就增中换行
		htmlText = strings.ReplaceAll(htmlText, "<", "\r\n<")
		htmlText = strings.ReplaceAll(htmlText, ">", ">\r\n")
		//根据reImg = `<img[\s\S]+?>` 正则提取数据
		value := GetValueFromHtml(reImg, htmlText)
		imgs := buildImg(value)
		urls := buildUrl("img", url, imgs)
		for _, v := range urls {
			if strings.Index(v, ".js") > 0 {
				continue
			}
			v = strings.ReplaceAll(v, `"`, ``)
			v = strings.ReplaceAll(v, "'", ``)
			v = strings.TrimSpace(v)
			//取得文件后缀
			suffix := ".jpg"
			m := strings.LastIndex(v, ".")
			if m != -1 {
				suffix = v[m:]
				//有些网页在扩展名加一些其它字符.要去掉.
				suffix = getImgSuffix(suffix)
			}
			fileName := DowloadImgPath + GetRandomName() + suffix

			img := imgInfo{
				//index:      imgNum,
				imgUrl:     v,
				fileName:   fileName,
				suffix:     suffix,
				isDownload: false,
			}
			linkImgs.Store(v, &img)
			fmt.Println("init img: ", url)
		}
	}
	imgWG.Done()
}

//初始化需要下载img的管道
func initImgChan() {
	i := 0
	linkImgs.Range(func(_, _ interface{}) bool {
		i++
		return true
	})
	imgChan = make(chan *imgInfo, i)
	imgFailed = make(chan *imgInfo, i)
	linkImgs.Range(func(_, v interface{}) bool {
		imgChan <- v.(*imgInfo)
		return true
	})
	close(imgChan)
}

//获取url的源码,返回*http.Response
func getUrlResp(url string) (resp *http.Response, err error) {
	client := http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err = client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("http get error,\n %s", err)
	}
	return resp, err
}

//判断是否有重复URL
func isContains(s []string, substr string) bool {
	for _, v := range s {
		if substr == v {
			return true
		}
	}
	return false
}

//取URL前面的头,是http还是https,因为有些URL链接,不加头.
func getUrlHead(url string) (head string) {
	i := strings.Index(url, "://")
	if i == -1 {
		head = "http"
		return
	}
	head = url[:i+1]
	return
}

//判断是否后缀是url
func isUrlSuffix(str string) (isUrl bool, url string) {
	//有些网页在扩展名加一些其它字符.要去掉.
	m := strings.LastIndex(str, ".")
	if m == -1 {
		return false, ""
	}
	prefix := str[:m]
	suffix := str[m:]
	switch suffix {
	case ".html":
		suffix = ".html"
	case ".htm":
		suffix = ".htm"
	case ".asp":
		suffix = ".asp"
	case ".aspx":
		suffix = ".aspx"
	case ".php":
		suffix = ".php"
	case ".jsp":
		suffix = ".jsp"
	case ".shtml":
		suffix = ".shtml"
	case ".shtm":
		suffix = ".shtm"
	default:
		return false, ""
	}
	url = prefix + suffix
	return true, url
}

//构建URL
func buildUrl(tag string, masterUrl string, value []string) (url []string) {
	if len(value) == 0 {
		return
	}
	url = []string{}
	head := getUrlHead(masterUrl)
	host := getHost(masterUrl)
	for i := 0; i < len(value); i++ {
		t := value[i]
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if len(t) < 5 {
			continue
		}
		//构建针对url的情况特殊处理．(不处理img的链接情况)
		if tag == "url" {
			isUrl, s := isUrlSuffix(t)
			if isUrl == false {
				continue
			}
			t = s
		}
		//img链接,根据情况添加链接的头.
		if strings.Contains(t, "http") || strings.Contains(t, "HTTP") {
			//判断是否有重复的URL
			if isContains(url, t) == false {
				url = append(url, t)
			}
			continue
		}
		if strings.Contains(t, `//`) {
			if isContains(url, head+t) == false {
				url = append(url, head+t)
			}
			continue
		}
		//取url的Path(不包括后面的文件名)
		var path string
		k := strings.LastIndex(masterUrl, `/`)
		if k != -1 {
			path = masterUrl[:k+1]
		}
		//判断前缀是否有/
		if !strings.HasPrefix(t, "/") {
			if path != "" {
				if isContains(url, path+t) == false {
					url = append(url, path+t)
				}
			}
		} else {
			if host != "" {
				if isContains(url, host+t) == false {
					url = append(url, host+t)
				}
			}
		}

	}
	return
}

//获取当前网页下的URL链接网址.
func getUrlLink(url string) (link []string) {
	//获取当前页面下的所以链接.
	HtmlText, err := getHtml(url)
	if err != nil {
		return nil
	}
	//如果没有换行,就增加换行
	HtmlText = strings.ReplaceAll(HtmlText, "<", "\r\n<")
	HtmlText = strings.ReplaceAll(HtmlText, ">", ">\r\n")
	//根据relinkALL正则提取数据
	value := GetValueFromHtml(relinkAll, HtmlText)
	temp := make([]string, len(value))
	for i, v := range value {
		temp[i] = v[1]
	}
	//从正则匹配到的数据中,构建url链接
	link = buildUrl("url", url, temp)
	return
}

//取img后缀,因为有些图片链接网址,会在后面加一些无关的字符.要去掉.
func getImgSuffix(s string) (v string) {
	n := strings.LastIndex(s, ".")
	if n == -1 {
		s = s + ".jpg"
		return s
	}
	prefix := s[:n]
	suffix := s[n:]
	if len(suffix) > 5 {
		suffix = ".jpg"
		s = prefix + suffix
	}
	return s
}

//从URl,以字符分割,得到字符之间的数据.
func cutUrl(s string, substr string) string {
	i := strings.Index(s, substr)
	if i >= 0 {
		s = s[i+len(substr):]
	}
	i = strings.Index(s, substr)
	if i >= 0 {
		s = s[:i]
	}
	return s
}

//尝试重新下载n次失败的.
func tryFailedImg() {
	close(imgFailed)
	if len(imgFailed) < 1 {
		return
	}
	fmt.Println("\ntry Dowload Failed img file. Please wait.")
	for v := range imgFailed {
		if v.isDownload == true {
			continue
		}
		Tryimg = append(Tryimg, *v)
	}
	if len(Tryimg) < 1 {
		return
	}

	for i := 0; i < TryImgNum; i++ {
		for n, img := range Tryimg {
			if img.isDownload == true {
				continue
			}
			start := time.Now()
			resp, err := getUrlResp(img.imgUrl)
			if err != nil {
				Tryimg[n].isDownload = false
				//fmt.Println("getUrlResp():", err)
				continue
			}
			defer resp.Body.Close()
			imgBytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				Tryimg[n].isDownload = false
				//fmt.Println("ioutil.ReadAll():", err)
				continue
			}

			mkdir(DowloadImgPath)

			err = ioutil.WriteFile(img.fileName, imgBytes, 0644)
			if err != nil {
				Tryimg[n].isDownload = false
				//fmt.Println("ioutil.WriteFile():", err)
				continue
			}
			Tryimg[n].isDownload = true
			cost := time.Since(start)
			fmt.Printf("try Download:%s Time:%s \n", img.fileName, cost)
		}
	}
}

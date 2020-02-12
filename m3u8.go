package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	TryNum = 3 //下载失败,再次重试下载次数.
)

var (
	DowloadM3u8Path = "./ts/"   //下载m3u8视频保存路径.当前目录下的ts目录.
	UrlInfo         = urlInfo{} //m3u8 url信息.
	TsUrls          = []ts{}    //ts url 具体下载ts片段文件的信息.
	TryTs           = []ts{}    //保存失败的ts片段文件信息,提供再次尝试下载
	TsChan          chan ts     //并发下载Ts文件的管道.
	TsFailed        chan ts     //存放下载ts失败的管道
)

//m3u8 url信息.
type urlInfo struct {
	Url           string //下载m3u8的URL
	Path          string //Url的前缀,不包含后面的文件名.
	Host          string //仅主域名.
	M3u8IndexFile string //M3u8的索引文件名.
	TsNum         int    //M3u8的索引文件中总ts文件数.
	IsEncryption  bool   //是否有加密
	Encryption    string //加密方式
	Key           []byte //解密KEY
	KeyUrl        string //获取解密KEY的URL
}

//ts文件的信息
type ts struct {
	index      int    //ts片段文件索引号.
	tsUrl      string //ts片段文件的URL下载地址.
	suffix     string //文件后缀
	isDownload bool   //是否下载成功
}

func init() {
	now := time.Now()
	formatNow := now.Format("20060102_15-04-05")
	DowloadM3u8Path = DowloadM3u8Path + formatNow + "/" //下载视频保存路径.
}

//下载m3u8视频
func dowloadM3u8(url string) (err error) {
	start := time.Now()

	//一.初始化需要下载的信息
	fmt.Println("Start initialization M3U8 url index. Please wait.")
	err = initAll(url)
	if err != nil {
		return err
	}

	//二.开始并发下载ts片段文件,默认设为15,可以命令行加-c调整.
	fmt.Println("Start dowload M3U8 url ts file.")
	for i := 0; i < maxGo; i++ {
		waitGroup.Add(1)
		go dowloadM3u8Go()
	}
	waitGroup.Wait()

	//三.尝试重新下载失败的,默认重试下载3次.
	tryFailed()

	//四.将ts片段文件合并为一个文件
	fmt.Println("\nStart Merge ts file, Please wait.")
	err = tsMerge()
	if err != nil {
		return
	}
	fmt.Println("\nDownload completed.")
	cost := time.Since(start)
	fmt.Printf("Total download time =[%s]\n", cost)
	return nil
}

//初始化所有信息,并将初始化数据存入管道
func initAll(url string) (err error) {
	//初始化基本信息
	err = initInfo(url)
	if err != nil {
		return err
	}
	//如果只有一条数据,则有下一层.
	//第一层M3U8中只是包含真的M3U8的路径,会重新再次获取.
	if len(TsUrls) == 1 {
		//取得第二层真正的m3mu文件的URL
		url = TsUrls[0].tsUrl
		err = initInfo(url)
		if err != nil {
			return err
		}
	}
	//初始化管道
	initChan()
	return nil
}

//初始化需要下载的基础信息.
func initInfo(url string) (err error) {
	//获取m3u8索引文件的URL的信息
	err = getUrlInfo(url)
	if err != nil {
		return err
	}
	//获取下载m3u8文件
	err = getM3u8(url)
	if err != nil {
		return err
	}
	//从下载的m3u8文件中,获取ts片段文件的信息.
	err = getTsUrls(DowloadM3u8Path + UrlInfo.M3u8IndexFile)
	if err != nil {
		return err
	}
	return nil
}

//获取m3u8索引文件的URL的信息
func getUrlInfo(url string) (err error) {
	i := strings.Index(url, `http`)
	if i == -1 {
		return fmt.Errorf("m3u8 URL is invalid")
	}
	host := getHost(url)
	if host == "" {
		return fmt.Errorf("m3u8 URL is invalid")
	}
	k := strings.LastIndex(url, `/`)
	if k == -1 {
		return fmt.Errorf("m3u8 URL is invalid")
	}
	UrlInfo = urlInfo{
		Url:           url,
		Path:          url[:k+1],
		Host:          host,
		M3u8IndexFile: url[k+1:],
	}
	return nil
}

//下载index.m3u8索引文件
func getM3u8(url string) (err error) {
	body, err := getUrl(url)
	if err != nil {
		return fmt.Errorf("m3u8 URL request failed:\n %w", err)
	}
	defer body.Close()

	err = getUrlInfo(url)
	if err != nil {
		return fmt.Errorf("Split Url: %s,\n %w", url, err)
	}

	bytes, err := ioutil.ReadAll(body)
	if err != nil {
		return fmt.Errorf("ioutil.ReadAll err: %s,\n %w", url, err)
	}

	err = mkdir(DowloadM3u8Path)
	if err != nil {
		return fmt.Errorf("mkdir err: %s,\n %w", url, err)
	}

	err = ioutil.WriteFile(DowloadM3u8Path+UrlInfo.M3u8IndexFile, bytes, 0644)
	if err != nil {
		return fmt.Errorf("get M3u8 index file failed: %s,\n %w", url, err)
	}
	return nil
}

//获取AES-128加密算法的KEY
func getKey(data string) (err error) {
	var n int
	var temp string
	switch {
	case strings.Contains(data, "URI"):
		n = strings.Index(data, "URI")
		temp = data[n+1:]
	case strings.Contains(data, "uri"):
		n = strings.Index(data, "uri")
		temp = data[n+1:]
	default:
		return fmt.Errorf("get AES128 key failed.")
	}

	i := strings.Index(temp, `=`)
	if i == -1 {
		return fmt.Errorf("get AES128 key failed.")
	}
	temp = temp[i+1:]
	temp = strings.Replace(temp, `"`, ``, -1)
	temp = strings.TrimSpace(temp)

	//生成要下载的完整URL
	j := strings.Index(temp, `/`)
	if j == -1 {
		temp = UrlInfo.Path + temp
	} else {
		temp = UrlInfo.Host + temp
	}
	//从URL中下载AES128解密KEY.
	body, err := getUrl(temp)
	if err != nil {
		return fmt.Errorf("get AES-128 key URL request failed:\n %w", err)
	}
	defer body.Close()
	bytes, err := ioutil.ReadAll(body)
	if err != nil {
		return fmt.Errorf("getKey(),ioutil.ReadAll err: %s,\n %w", err)
	}
	if len(bytes) > 0 {
		UrlInfo.IsEncryption = true
		UrlInfo.Encryption = "AES-128"
		UrlInfo.Key = bytes
		UrlInfo.KeyUrl = temp
	}
	return nil
}

//从下载的m3u8文件中,组成各ts文件完整的URL
func getTsUrls(fileName string) (err error) {
	var num int = 0
	f, err := os.OpenFile(fileName, os.O_RDONLY, 0)
	if err != nil {
		err = fmt.Errorf("open file(%s)failed:%w\n", fileName, err)
		return err
	}
	defer f.Close()
	fileScanner := bufio.NewScanner(f)
	for fileScanner.Scan() {
		line := fileScanner.Text()
		line = strings.TrimSpace(line)
		// 以#或;开头视为注释,空行和注释不读取
		if line == "" {
			continue
		}
		//判断是否有AES-128加密.有加密就获取KEY.
		if strings.HasPrefix(line, "#EXT-X-KEY") && strings.Contains(line, "AES-128") {
			//获取AES-128加密key和KeyURL
			err := getKey(line)
			if err != nil {
				fmt.Println(err)
				err = nil
			}
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, ";") {
			continue
		}
		if strings.Contains(line, `//`) && !strings.HasPrefix(line, `//`) {
			line = strings.ReplaceAll(line, `//`, `/`)
		}

		var tsUrl string
		var suffix string
		//给TS文件加后缀
		i := strings.LastIndex(line, ".")
		if i == -1 {
			suffix = ".ts"
		} else {
			suffix = line[i:]
		}
		//生成完整的URL,是主域名+路径,或是前缀路径+文件名
		j := strings.Index(line, `/`)
		if j == -1 {
			tsUrl = UrlInfo.Path + line
		} else {
			tsUrl = UrlInfo.Host + line
		}
		//如果是完整的路径,直接赋值.
		if strings.Contains(line, "http") {
			tsUrl = line
		}
		//如是第一层M3U8中只是包含真正的M3U8的路径,会重新再次获取,之前第一个已赋值.
		//判断是否是第二层.是就从0开始赋值.
		if num == 0 && len(TsUrls) == 1 {
			TsUrls[0].index = num
			TsUrls[0].tsUrl = tsUrl
			TsUrls[0].suffix = suffix
			TsUrls[0].isDownload = false
		} else {
			ts := ts{
				index:      num,
				tsUrl:      tsUrl,
				suffix:     suffix,
				isDownload: false,
			}
			TsUrls = append(TsUrls, ts)
		}
		num++
	}
	UrlInfo.TsNum = num

	return nil
}

//初始化管道,把需要下载的放入管道.
func initChan() {
	TsChan = make(chan ts, UrlInfo.TsNum)
	TsFailed = make(chan ts, UrlInfo.TsNum)
	for i, ts := range TsUrls {
		if i <= UrlInfo.TsNum {
			TsChan <- ts
		} else {
			fmt.Println("initchan():Ts file is not equal ")
		}
	}
	close(TsChan)
}

//合并ts文件
func tsMerge() (err error) {
	//视频汇总文件,文件名随机生成,保存在当前目录
	rand.Seed(time.Now().UnixNano())
	var MergeFile string
	for {
		MergeFile = "movie" + strconv.Itoa(rand.Intn(1000)) + ".ts"
		if !fileExists(MergeFile) {
			break
		}
	}
	file, err := os.Create(MergeFile)
	if err != nil {
		return fmt.Errorf("Create merge file failed：%s", err)
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	count := 0

	//从DowloadPath的临时下载目录中,读取各TS片段文件.进行合并.
	for _, ts := range TsUrls {
		tsPath := DowloadM3u8Path + strconv.Itoa(ts.index) + ts.suffix

		bytes, err := ioutil.ReadFile(tsPath)
		if err != nil {
			continue
		}
		_, err = writer.Write(bytes)
		if err != nil {
			continue
		}

		count++
	}
	err = writer.Flush()
	if err != nil {
		return fmt.Errorf("merge file failed：%s", err)
	}

	//删除临时下载的TS片段目录
	err = os.RemoveAll(DowloadM3u8Path)
	if err != nil {
		return fmt.Errorf("delete temp dowload file failed：%s", err)
	}
	if count != UrlInfo.TsNum {
		return fmt.Errorf("[warning] %d Missing ts file download:", UrlInfo.TsNum-count)
	}

	return nil
}

//并发下载Ts片段文件
func dowloadM3u8Go() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("dowload panic", err)
			waitGroup.Done()
		}
	}()
	//从TS管道中,读取下载各单个TS文件,将失败的放入Failed管道中.
	for ts := range TsChan {
		start := time.Now()
		if ts.isDownload == true {
			continue
		}
		body, err := getUrl(ts.tsUrl)
		if err != nil {
			ts.isDownload = false
			TsFailed <- ts
			fmt.Println("getUrl():", err)
			continue
		}
		defer body.Close()
		fileName := strconv.Itoa(ts.index) + ts.suffix
		bytes, err := ioutil.ReadAll(body)
		if err != nil {
			ts.isDownload = false
			TsFailed <- ts
			fmt.Println("ioutil.ReadAll():", fileName, err)
			continue
		}
		//解密 AES-128
		if UrlInfo.IsEncryption && UrlInfo.Encryption == "AES-128" {
			temp, err := DecryptAES128(bytes, UrlInfo.Key)
			if err != nil {
				fmt.Println(err)
			}
			bytes = temp
		}

		Path := DowloadM3u8Path + fileName
		err = ioutil.WriteFile(Path, bytes, 0644)
		if err != nil {
			ts.isDownload = false
			TsFailed <- ts
			fmt.Println("ioutil.WriteFile():", Path, err)
			continue
		}
		ts.isDownload = true
		cost := time.Since(start)
		fmt.Printf(" Download:%s  Time:%s \n", Path, cost)
	}
	waitGroup.Done()
}

//尝试重新下载失败的,TryNum=3再次重试3次.
func tryFailed() {
	close(TsFailed)
	if len(TsFailed) < 1 {
		return
	}
	fmt.Println("\ntry dowload Failed M3U8 url ts file. Please wait.")
	for ts := range TsFailed {
		if ts.isDownload == true {
			continue
		}
		TryTs = append(TryTs, ts)
	}
	if len(TryTs) < 1 {
		return
	}

	for i := 0; i < TryNum; i++ {
		for n, ts := range TryTs {
			if ts.isDownload == true {
				continue
			}
			body, err := getUrl(ts.tsUrl)
			if err != nil {
				TryTs[n].isDownload = false
				fmt.Println("getUrl():", err)
				continue
			}
			defer body.Close()
			fileName := strconv.Itoa(ts.index) + ts.suffix
			bytes, err := ioutil.ReadAll(body)
			if err != nil {
				TryTs[n].isDownload = false
				fmt.Println("ioutil.ReadAll():", fileName, err)
				continue
			}
			Path := DowloadM3u8Path + fileName
			err = ioutil.WriteFile(Path, bytes, 0644)
			if err != nil {
				TryTs[n].isDownload = false
				fmt.Println("ioutil.WriteFile():", Path, err)
				continue
			}
			TryTs[n].isDownload = true
			fmt.Println("Try Download:", Path)
		}
	}
}

package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/axgle/mahonia"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	mutex sync.Mutex //互斥锁
)
func  get_html(url string) string {
	header := map[string]string{
		"Host": "movie.douban.com",
		"Connection": "keep-alive",
		"Cache-Control": "max-age=0",
		"Upgrade-Insecure-Requests": "1",
		"User-Agent": "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/53.0.2785.143 Safari/537.36",
		"Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"Referer": "https://movie.douban.com/top250",
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
	}
	for k, v := range header {
		req.Header.Add(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
	}
	return string(body)

}
//获取url的数据
func getUrl(url string) (io.ReadCloser, error) {
	client := http.Client{
		Timeout: 90 * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("http get error,\n %s", err)
	}
	return resp.Body, nil
}

//根据url得到主域名
func getHost(url string) (host string) {
	i := strings.LastIndex(url, `//`)
	if i == -1 {
		host = ""
		return
	}
	temp := url[i+2:]
	host = url[:i+2]
	i = strings.Index(temp, `/`)
	if i == -1 {
		host = ""
		return
	}
	temp = temp[:i]
	host = host + temp
	return
}

//根据正则表达式规则和网址，在网页源码中查找相应的数据（第一个参数为正则表达式，第二个为网址）
func GetValueFromHtml(regexpStr, html string) [][]string {
	re := regexp.MustCompile(regexpStr)
	allString := re.FindAllStringSubmatch(html, -1)
	return allString
}

//根据正则,判断是否有数据.
func isRegExists(regexpStr, data string) bool {
	re := regexp.MustCompile(regexpStr)
	allString := re.FindAllStringSubmatch(data, -1)
	if len(allString) > 0 {
		return true
	}
	return false
}

//判断文件是否存在
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

//判断目录是否存在
func pathExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

//创建目录，如果没有就创建。
func mkdir(dir string) (err error) {
	exist, err := pathExist(dir)
	if err != nil {
		return fmt.Errorf("get dir error!: %s", err)
	}
	if !exist {
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("mkdir failed![%v]\n", err)
		}
	}
	return nil
}

//AES128解密
func DecryptAES128(data, key []byte) ([]byte, error) {
	if len(key) < 1 {
		err := fmt.Errorf("Not a AES(128) Key")
		return nil, err
	}
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("DecryptAES128 panic:", err)
		}
	}()
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()

	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	result := make([]byte, len(data))
	blockMode.CryptBlocks(result, data)
	result = PKCS7UnPadding(result)
	return result, nil
}

//去补码
func PKCS7UnPadding(data []byte) []byte {
	length := len(data)
	unpadding := int(data[length-1])
	return data[:length-unpadding]
}

//转换字符编码
func ConvertToString(src string, srcCode string, tagCode string) string {
	srcCoder := mahonia.NewDecoder(srcCode)
	srcResult := srcCoder.ConvertString(src)
	tagCoder := mahonia.NewDecoder(tagCode)
	_, cdata, _ := tagCoder.Translate([]byte(srcResult), true)
	result := string(cdata)
	return result
}

//得到[start,end]之间的随机整数,加锁，1纳秒执行。
func GetRanddomInt(start, end int) int {
	mutex.Lock()
	<-time.After(1 * time.Nanosecond)                    //延时1纳秒
	r := rand.New(rand.NewSource(time.Now().UnixNano())) //根据时间戳生成随机数
	v := start + r.Intn(end-start)
	mutex.Unlock()
	return v
}

//根据时间戳,生成随机数文件名.
func GetRandomName() string {
	timestamp := strconv.Itoa(int(time.Now().UnixNano()))
	randomNum := strconv.Itoa(GetRanddomInt(1, 1000))
	return timestamp + "_" + randomNum
}

//得到网站的字符集
func GetUrlCharacterSet(url string) (charSet string, err error) {
	//reCharacterSet=`<meta.+?charset=(.+?)"`
	html, err := getHtml(url)
	if err != nil {
		return "", err
	}
	allString := GetValueFromUrl(reCharacterSet, html)
	for _, v := range allString {
		charSet = v[1]
	}
	return charSet, err
}

//得到body,Html文本中的字符集
func GetHtmlCharacterSet(HtmlText string) string {
	//reCharacterSet=`<meta.+?charset=(.+?)"`
	re := regexp.MustCompile(reCharacterSet)
	allString := re.FindAllStringSubmatch(HtmlText, 1)
	var charSet string
	for _, v := range allString {
		//charSet=v[1]
		charSet = strings.Replace(v[1], `"`, "", -1)
		charSet = strings.Replace(charSet, " ", "", -1)
	}

	return charSet
}

//根据网址，得到整个网页的源码
func getHtml(url string) (html string, err error) {
	body, err := getUrl(url)
	if err != nil {
		err = fmt.Errorf("getUrl():", err)
		return "", err
	}
	defer body.Close()
	bytes, err := ioutil.ReadAll(body)
	if err != nil {
		err = fmt.Errorf("ioutil.ReadAll():", err)
		return "", err
	}

	html = string(bytes)
	//html = ConvertToString(html, "gbk", "utf-8")
	return html, err
}

//根据正则表达式规则和网址，在网页源码中查找相应的数据（第一个参数为正则表达式，第二个为网址源码）
func GetValueFromUrl(regexpStr string, html string) [][]string {
	re := regexp.MustCompile(regexpStr)
	allString := re.FindAllStringSubmatch(html, -1)
	return allString
}

//封装Go语言原生的http库，发送可以带参数、请求体、并可以定制http请求头部
//Get http get method
func Get(url string, params map[string]string, headers map[string]string) (*http.Response, error) {
	//new request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err)
		return nil, errors.New("new request is fail ")
	}
	//add params
	q := req.URL.Query()
	if params != nil {
		for key, val := range params {
			q.Add(key, val)
		}
		req.URL.RawQuery = q.Encode()
	}
	//add headers
	if headers != nil {
		for key, val := range headers {
			req.Header.Add(key, val)
		}
	}
	//http client
	client := &http.Client{}
	log.Printf("Go GET URL : %s \n", req.URL.String())
	return client.Do(req)
}

//Post http post method
func Post(url string, body map[string]string, params map[string]string, headers map[string]string) (*http.Response, error) {
	//add post body
	var bodyJson []byte
	var req *http.Request
	if body != nil {
		var err error
		bodyJson, err = json.Marshal(body)
		if err != nil {
			log.Println(err)
			return nil, errors.New("http post body to json failed")
		}
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyJson))
	if err != nil {
		log.Println(err)
		return nil, errors.New("new request is fail: %v \n")
	}
	req.Header.Set("Content-type", "application/json")
	//add params
	q := req.URL.Query()
	if params != nil {
		for key, val := range params {
			q.Add(key, val)
		}
		req.URL.RawQuery = q.Encode()
	}
	//add headers
	if headers != nil {
		for key, val := range headers {
			req.Header.Add(key, val)
		}
	}
	//http client
	client := &http.Client{}
	log.Printf("Go POST URL : %s \n", req.URL.String())
	return client.Do(req)
}

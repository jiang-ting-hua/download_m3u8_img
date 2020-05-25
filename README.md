### 一,根据URL的m3u8索引,下载视频文件.
    增加二种UI图形界面在这二个地址:
    https://github.com/jiang-ting-hua/example/tree/master/durl2  这种是HTML的图型界面 github.com/sciter-sdk/go-sciter
    https://github.com/jiang-ting-hua/example/tree/master/durl3  这种是利用 github.com/andlabs/ui 的图形界面

### 二,下载网页中的图片.

为了家里小朋友,要下载一个视频,放在电视上看. 所以利用一点时间,写了这个简单下载程序.如果视频有加密,会对其解密.
最近下点图片,又增加下载图片的功能.下载当前页(包含当前页中的网址链接中的图片).

获取m3u8索引的URL方法:
需要用google或360浏览器,进入开发者模式,按F12或ctrl+shift+c,在里面点network,再把网页刷新,搜索m3u8,就可找到index.m3u8文件的URL

以下是参数,下载失败的ts会重试下载三次,图片只重试二次:

-m  用于视频,要下载的index.m3u8网址.

-i  用于图片,要下载的图片URL.

-s  (可以不输入)用于图片,下载大于该值大小的图片,默认下载大于30KB的图片.例如:(-s 80)只下载大于80KB的图片.

-l  (可以不输入)用于图片,以当前参数i中输入的为第一层,向下搜索几层.默认向下搜索3层.(当以主页为第一层,层数加大,可以全站下载图片,时间要久点)

-c  (可以不输入)(用于图片和视频),并发数量,默认是15个并发.
(并发数量可以加大,这样可以加快下载速度.但考虑视频网站的压力,别设大了,温柔下载)

例如:

### 下载视频:

简单用法:   dmi.exe -m https://www.mmicloud.com:65/20191204/I2jpA2LP/index.m3u8

加参数用法: dmi.exe -c 20 -m https://www.mmicloud.com:65/20191204/I2jpA2LP/index.m3u8

### 下载图片:

简单用法:   dmi.exe -i https://www.abc.com/

加参数用法: dmi.exe -c 20 -s 80 -l 5 -i https://www.abc.com/

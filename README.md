# dowload_m3u8
为了家里小朋友,要下载一个视频,放在电视上看. 所以利用一点时间,写了这个简单下载程序.如果视频有加密,会对其解密.

获取m3u8索引的URL方法:
需要用google或360浏览器,进入开发者模式,按F12或ctrl+shift+c,在里面点network,再把网页刷新,搜索m3u8,就可找到index.m3u8文件的URL

只有二个参数,下载失败的ts会重试下载三次:

-u 要下载的index.m3u8网址.

-c 并发数量,默认是15.可以不输入.
(并发数量可以加大,这样可以加快下载速度.但考虑视频网站的压力,别设大了,温柔下载)

例如:

dm3u8.exe -u https://cn4.qxreader.com/hls/20200131/baeee825f6605d5ab28b954f07e24386/1580471232/index.m3u8

dm3u8.exe -c 30 -u https://www.mmicloud.com:65/20191204/I2jpA2LP/index.m3u8

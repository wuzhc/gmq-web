> gmq的web管理系统;你可以在web页面中,手动注册注销节点,查看各个topic统计信息,修改topic配置信息,模拟推送拉取消息

## 使用
### 安装`glide`
安装依赖管理工具`glide`,如果你已经安装了,可以跳过这步
```
git clone https://github.com/xkeyideal/glide.git $GOPATH/src/github.com/xkeyideal/glide
cd $GOPATH/src/github.com/xkeyideal/glide
make install

# 如果你翻不了墙,可以将`golang.org/x/sys/unix`映射到`github.com/golang/sys`上,在终端上执行下面命令
glide mirror set https://golang.org/x/sys/unix https://github.com/golang/sys --base golang.org/x/sys
```

### 安装`gmq-web`
```bash
git clone https://github.com/wuzhc/gmq-web.git $GOPATH/src/github.com/wuzhc/gmq-web
cd $GOPATH/src/github.com/wuzhc/gmq-web
make

# 启动服务
gweb -web_addr="127.0.0.1:8080" -register_addr="http://127.0.0.1:9595"
# 或者直接运行源码
go run main.go -web_addr="127.0.0.1:8080" -register_addr="http://127.0.0.1:9595"
```
参数说明:
- web_addr `gmq-web`监听的ip和端口号,默认端口为`8080`
- register_addr 注册中心的http地址

### docker运行
```bash
docker run --name gmq-web -p 8080:8080 wuzhc/gmq-image:v1 gweb -web_addr="127.0.0.1:8080" -register_addr="http://127.0.0.1:9595"
```

## 访问
游览器输入地址:`http://127.0.0.0:8080`
### 节点列表
![节点列表](https://gitee.com/wuzhc123/zcnote/raw/master/images/gmq/gmq-web%E8%8A%82%E7%82%B9%E5%88%97%E8%A1%A8.png)
### topic列表
![topic列表](https://gitee.com/wuzhc123/zcnote/raw/master/images/gmq/gmq-web%E4%B8%BB%E9%A2%98%E5%88%97%E8%A1%A8.png)
### 消息处理
![推送](https://gitee.com/wuzhc123/zcnote/raw/master/images/gmq/gmq-web%E6%B6%88%E6%81%AF%E6%8E%A8%E9%80%81.png)
![消费](https://gitee.com/wuzhc123/zcnote/raw/master/images/gmq/gmq-web%E6%B6%88%E6%81%AF%E6%B6%88%E8%B4%B9.png)
![确认](https://gitee.com/wuzhc123/zcnote/raw/master/images/gmq/gmq-web%E6%B6%88%E6%81%AF%E7%A1%AE%E8%AE%A4.png)

后台模板来自https://github.com/george518/PPGo_Job
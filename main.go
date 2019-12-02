package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/etcd-io/etcd/clientv3"
	"github.com/gin-gonic/gin"
)

type node struct {
	Id       string `json:"node_id"`
	TcpAddr  string `json:"tcp_addr"`
	HttpAddr string `json:"http_addr"`
	Weight   string `json:"weight"`
	JoinTime string `json:"join_time"`
}

type topicData struct {
	Name      string `json:"name"`
	PopNum    int64  `json:"pop_num"`
	PushNum   int64  `json:"push_num"`
	BucketNum int    `json:"bucket_num"`
	DeadNum   int    `json:"dead_num"`
	StartTime string `json:"start_time"`
}

type respStruct struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
}

type msg struct {
	Topic    string `json:"topic"`
	Body     string `json:"body"`
	Delay    int    `json:"delay"`
	routeKey string `json:"route_key"`
}

var webAddr string
var etcdCli *clientv3.Client
var registerAddr string

func main() {
	// parse command options
	var endpoints string
	flag.StringVar(&endpoints, "ectd_endpoints", "127.0.0.1:2379", "etcd endpoints")
	flag.StringVar(&webAddr, "web_addr", ":8080", "the address of gmq-web")
	flag.Parse()

	// connect to etcd
	ectdEndpoints := strings.Split(endpoints, ",")
	err := connectToEtcd(ectdEndpoints)
	if err != nil {
		log.Fatalf("connect to etcd failed, %s", err)
	}

	// run gin
	var ctx context.Context
	ctx = context.Background()
	ctx, cancel := context.WithCancel(ctx)
	run(ctx, cancel)
}

func connectToEtcd(endpoints []string) error {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("can't create etcd client.")
	}

	etcdCli = cli
	return nil
}

func run(ctx context.Context, cancel context.CancelFunc) {
	// gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.StaticFS("/static", http.Dir("static"))
	r.LoadHTMLGlob("views/*")
	r.GET("/", index)
	r.GET("/login", login)
	r.GET("/home", home)

	// 节点管理
	r.GET("/nodeList", nodeList)
	r.GET("/addNode", addNode)
	r.GET("/getNodes", getNodes)
	r.POST("/registerNode", registerNode)
	r.GET("/unRegisterNode", unRegisterNode)
	r.POST("/editNodeWeight", editNodeWeight)

	// 主题topic管理
	r.GET("/topicList", topicList)
	r.GET("/removeTopic", removeTopic)
	r.GET("/getTopics", getTopics)
	r.GET("/setIsAutoAck", setIsAutoAck)

	// 消息管理
	r.GET("/msgDemo", msgDemo)
	r.GET("/declare", declareQueue)
	r.POST("/push", push)
	r.GET("/pop", pop)
	r.GET("/ack", ack)
	// r.GET("/dead", dead)
	// r.GET("/mpush", mpush)

	serv := &http.Server{
		Addr:    webAddr,
		Handler: r,
	}

	go func() {
		<-ctx.Done()
		if err := serv.Shutdown(ctx); err != nil {
			log.Fatalln("web exit:", err)
		}
		log.Println("web exist")
	}()

	if err := serv.ListenAndServe(); err != nil {
		return
	}
}

// 首页
func index(c *gin.Context) {
	c.HTML(http.StatusOK, "entry.html", gin.H{
		"siteName":      "gmq-web管理",
		"version":       "v3.0",
		"loginUserName": "wuzhc",
	})
}

// 主页
func home(c *gin.Context) {
	c.HTML(http.StatusOK, "home.html", gin.H{
		"title": "主页",
	})
}

// 登录页
func login(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{
		"title": "登录页面",
	})
}

// 节点管理页面
func nodeList(c *gin.Context) {
	c.HTML(http.StatusOK, "node_list.html", gin.H{
		"title": "节点管理",
	})
}

// 注册节点页面
func addNode(c *gin.Context) {
	c.HTML(http.StatusOK, "add_node.html", gin.H{
		"title": "注册节点",
	})
}

// topic列表
func topicList(c *gin.Context) {
	nodes, err := _getNodes()
	if err != nil {
		c.HTML(http.StatusBadGateway, "error.html", gin.H{
			"error": err,
		})
		return
	}

	c.HTML(http.StatusOK, "topic_list.html", gin.H{
		"title": "topic管理",
		"nodes": nodes,
	})
}

// 获取正在运行的topic统计信息
func getTopics(c *gin.Context) {
	addr := c.Query("addr")
	if len(addr) == 0 {
		c.JSON(http.StatusBadRequest, rspErr("please select a node."))
		return
	}

	gmqApi("get", "http://"+addr+"/getAllTopicStat", nil, c)
}

// 删除topic
func removeTopic(c *gin.Context) {
	addr := c.Query("addr")
	topic := c.Query("topic")
	if len(addr) == 0 {
		c.JSON(http.StatusBadRequest, rspErr("please select a node."))
		return
	}
	if len(topic) == 0 {
		c.JSON(http.StatusBadRequest, rspErr("topic is empty."))
		return
	}

	v := url.Values{}
	v.Set("topic", topic)
	gmqApi("get", "http://"+addr+"/exitTopic", v, c)
}

func setIsAutoAck(c *gin.Context) {
	addr := c.Query("addr")
	topic := c.Query("topic")
	if len(addr) == 0 {
		c.JSON(http.StatusBadRequest, rspErr("please select a node."))
		return
	}
	if len(topic) == 0 {
		c.JSON(http.StatusBadRequest, rspErr("topic is empty."))
		return
	}

	v := url.Values{}
	v.Set("topic", topic)
	gmqApi("get", "http://"+addr+"/setIsAutoAck", v, c)
}

// 获取注册中心所有注册节点
func getNodes(c *gin.Context) {
	nodes, err := _getNodes()
	if err != nil {
		c.JSON(http.StatusBadGateway, err)
		return
	}

	var rspData respStruct
	rspData.Data = nodes
	c.JSON(http.StatusOK, rspData)
}

// 注销节点
func unRegisterNode(c *gin.Context) {
	addr := c.Query("addr")
	if len(addr) == 0 {
		c.JSON(http.StatusBadRequest, rspErr("addr is empty"))
		return
	}

	nodes, err := _getNodes()
	if err != nil {
		c.JSON(http.StatusBadGateway, rspErr(err.Error()))
		return
	}

	var nodeKey string
	for k, n := range nodes {
		if n.TcpAddr == addr {
			nodeKey = k
			break
		}
	}
	if len(nodeKey) == 0 {
		c.JSON(http.StatusBadGateway, rspErr("addr can't match node."))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	_, err := etcdCli.Get(ctx, nodeKey)
	cancel()
	if err != nil {
		c.JSON(http.StatusBadGateway, rspErr(err.Error()))
		return
	}

	c.JSON(http.StatusOK, rspSuccess("success"))
}

// 注册节点
func registerNode(c *gin.Context) {
	tcpAddr := c.PostForm("tcp_addr")
	if len(tcpAddr) == 0 {
		c.JSON(http.StatusBadRequest, rspErr("tcp_add is empty"))
		return
	}
	httpAddr := c.PostForm("http_addr")
	if len(httpAddr) == 0 {
		c.JSON(http.StatusBadRequest, rspErr("http_add is empty"))
		return
	}
	weight := c.PostForm("weight")
	if len(weight) == 0 {
		c.JSON(http.StatusBadRequest, rspErr("weight must be greater than 0."))
		return
	}
	id := c.PostForm("node_id")
	nodeId, _ := strconv.Atoi(id)
	if nodeId > 1024 || nodeId < 1 {
		c.JSON(http.StatusBadRequest, rspErr("node_id must be between 1 and 1024."))
		return
	}

	gmqApi("get", registerAddr+"/register?node_id="+id+"&tcp_addr="+tcpAddr+"&http_addr="+httpAddr+"&weight="+weight, nil, c)
}

// 修改节点权重
func editNodeWeight(c *gin.Context) {
	tcpAddr := c.PostForm("addr")
	if len(tcpAddr) == 0 {
		c.JSON(http.StatusBadRequest, rspErr("tcp_add is empty"))
		return
	}
	weight := c.PostForm("weight")
	if len(weight) == 0 {
		c.JSON(http.StatusBadRequest, rspErr("weight must be greater than 0."))
		return
	}

	gmqApi("get", registerAddr+"/editWeight?tcp_addr="+tcpAddr+"&weight="+weight, nil, c)
}

// 消息测试
func msgDemo(c *gin.Context) {
	nodes, err := _getNodes()
	if err != nil {
		c.HTML(http.StatusBadGateway, "error.html", gin.H{
			"error": err,
		})
		return
	}

	c.HTML(http.StatusOK, "msg_demo.html", gin.H{
		"title": "消息测试",
		"nodes": nodes,
	})
}

// 推送消息
func push(c *gin.Context) {
	addr := c.PostForm("addr")
	topic := c.PostForm("topic")
	content := c.PostForm("content")
	routeKey := c.PostForm("routeKey")
	delay := c.DefaultPostForm("delay", "0")
	if len(addr) == 0 {
		c.JSON(http.StatusBadRequest, "please select a node.")
		return
	}
	if len(topic) == 0 {
		c.JSON(http.StatusBadRequest, "topic is empty")
		return
	}
	if len(content) == 0 {
		c.JSON(http.StatusBadRequest, "content is empty")
		return
	}
	if len(routeKey) == 0 {
		c.JSON(http.StatusBadRequest, "routeKey is empty")
		return
	}

	m := msg{}
	m.Topic = topic
	m.Body = content
	m.Delay, _ = strconv.Atoi(delay)
	m.routeKey = routeKey
	data, err := json.Marshal(m)
	if err != nil {
		c.JSON(http.StatusBadGateway, "encode message failed.")
		return
	}

	v := url.Values{}
	v.Set("data", string(data))
	gmqApi("POST", "http://"+addr+"/push", v, c)
}

// 消费消息
// curl "http://127.0.0.1:9504/pop?topic=xxx&bindKey=xxx"
func pop(c *gin.Context) {
	topic := c.Query("topic")
	addr := c.Query("addr")
	bindKey := c.Query("bindKey")
	if len(topic) == 0 {
		c.JSON(http.StatusBadRequest, "topic is empty")
		return
	}
	if len(addr) == 0 {
		c.JSON(http.StatusBadRequest, "please select a node.")
		return
	}
	if len(bindKey) == 0 {
		c.JSON(http.StatusBadRequest, "bindKey is empty.")
		return
	}

	v := url.Values{}
	v.Set("topic", topic)
	v.Set("bindKey", bindKey)
	gmqApi("get", "http://"+addr+"/pop", v, c)
}

// 声明队列
// curl "http://127.0.0.1:9504/declareQueue?topic=xxx&bindKey=kkk"
func declareQueue(c *gin.Context) {
	addr := c.Query("addr")
	bindKey := c.Query("bindKey")
	topic := c.Query("topic")
	if len(addr) == 0 {
		c.JSON(http.StatusBadRequest, "addr is empty")
		return
	}
	if len(bindKey) == 0 {
		c.JSON(http.StatusBadRequest, "bindKey is empty")
		return
	}
	if len(topic) == 0 {
		c.JSON(http.StatusBadRequest, "topic is empty")
		return
	}

	v := url.Values{}
	v.Set("topic", topic)
	v.Set("bindKey", bindKey)
	gmqApi("get", "http://"+addr+"/declareQueue", v, c)
}

// 确认消息
func ack(c *gin.Context) {
	addr := c.Query("addr")
	msgId := c.Query("msgId")
	topic := c.Query("topic")
	bindKey := c.Query("bindKey")
	if len(addr) == 0 {
		c.JSON(http.StatusBadRequest, "addr is empty")
		return
	}
	if len(msgId) == 0 {
		c.JSON(http.StatusBadRequest, "msgId is empty")
		return
	}
	if len(topic) == 0 {
		c.JSON(http.StatusBadRequest, "topic is empty")
		return
	}
	if len(bindKey) == 0 {
		c.JSON(http.StatusBadRequest, "bindKey is empty")
		return
	}

	v := url.Values{}
	v.Set("topic", topic)
	v.Set("msgId", msgId)
	v.Set("bindKey", bindKey)
	gmqApi("get", "http://"+addr+"/ack", v, c)
}

// 批量推送消息
func mpush(c *gin.Context) {
	c.JSON(http.StatusOK, "unsport")
}

func gmqApi(method string, addr string, data url.Values, c *gin.Context) {
	client := &http.Client{}
	method = strings.ToUpper(method)

	var (
		req *http.Request
		err error
	)

	if method == "POST" {
		req, err = http.NewRequest(method, addr, strings.NewReader(data.Encode()))
		if err != nil {
			c.JSON(http.StatusBadGateway, rspErr(err))
			return
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else if method == "GET" {
		req, err = http.NewRequest("GET", addr+"?"+data.Encode(), nil)
		if err != nil {
			c.JSON(http.StatusBadGateway, rspErr(err))
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, rspErr("unkown request method."))
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, rspErr(err))
		return
	}
	if resp.StatusCode != 200 {
		c.JSON(resp.StatusCode, rspErr("request failed."))
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusBadGateway, rspErr(err))
		return
	}

	var rspData respStruct
	if err := json.Unmarshal(body, &rspData); err != nil {
		c.JSON(http.StatusBadGateway, rspErr(err))
		return
	}

	c.JSON(http.StatusOK, rspData)
}

// 获取节点
func _getNodes() (map[string]node, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	resp, err := etcdCli.Get(ctx, "/gmq/node", clientv3.WithPrefix())
	cancel()
	if err != nil {
		return nil, err
	}

	var n node
	nodes := make(map[string]node)
	for _, ev := range resp.Kvs {
		fmt.Printf("%s => %s\n", ev.Key, ev.Value)
		if err := json.Unmarshal(ev.Value, &n); err != nil {
			return nil, err
		}
		nodes[string(ev.Key)] = n
	}

	return nodes, nil
}

func rspErr(msg interface{}) gin.H {
	var resp = make(gin.H)
	resp["code"] = 1
	resp["msg"] = msg
	resp["data"] = nil
	return resp
}

func rspData(data interface{}) gin.H {
	var resp = make(gin.H)
	resp["code"] = 0
	resp["msg"] = ""
	resp["data"] = data
	return resp
}

func rspSuccess(msg interface{}) gin.H {
	var resp = make(gin.H)
	resp["code"] = 0
	resp["msg"] = msg
	resp["data"] = nil
	return resp
}

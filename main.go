package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	API_GMQ_NODE     = "http://127.0.0.1:9504"
	API_GMQ_REGISTER = "http://127.0.0.1:9595"
)

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
	Topic string `json:"topic"`
	Body  string `json:"body"`
	Delay int    `json:"delay"`
}

func main() {
	var ctx context.Context
	ctx = context.Background()
	ctx, cancel := context.WithCancel(ctx)
	run(ctx, cancel)
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
	r.GET("/getTopic", getTopic)
	r.GET("/getTopics", getTopics)
	r.GET("/setIsAutoAck", setIsAutoAck)

	// 消息管理
	r.GET("/msgDemo", msgDemo)
	r.GET("/msgPush", msgPush)
	r.GET("/msgPop", msgPop)
	r.GET("/msgAck", msgAck)
	r.GET("/msgDead", msgDead)
	r.POST("/push", push)
	r.GET("/pop", pop)
	r.GET("/ack", ack)
	// r.GET("/dead", dead)
	// r.GET("/mpush", mpush)

	serv := &http.Server{
		Addr:    ":8585",
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
	data, err := _getNodes()
	if err != nil {
		c.HTML(http.StatusBadGateway, "error.html", gin.H{
			"error": err,
		})
		return
	}

	c.HTML(http.StatusOK, "topic_list.html", gin.H{
		"title": "topic管理",
		"data":  data,
	})
}

// 获取topic
func getTopic(c *gin.Context) {
	topic := c.Query("topic")
	if len(topic) == 0 {
		c.JSON(http.StatusBadRequest, rspErr("topic is empty"))
		return
	}

	v := url.Values{}
	v.Set("topic", topic)
	gmqApi("get", API_GMQ_NODE+"/getTopic", v, c)
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
	gmqApi("get", API_GMQ_REGISTER+"/getNodes", nil, c)
}

// 注销节点
func unRegisterNode(c *gin.Context) {
	addr := c.Query("addr")
	if len(addr) == 0 {
		c.JSON(http.StatusBadRequest, rspErr("addr is empty"))
		return
	}

	gmqApi("get", API_GMQ_REGISTER+"/unregister?tcp_addr="+addr, nil, c)
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

	gmqApi("get", API_GMQ_REGISTER+"/register?node_id="+id+"&tcp_addr="+tcpAddr+"&http_addr="+httpAddr+"&weight="+weight, nil, c)
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

	gmqApi("get", API_GMQ_REGISTER+"/editWeight?tcp_addr="+tcpAddr+"&weight="+weight, nil, c)
}

// 消息测试
func msgDemo(c *gin.Context) {
	c.HTML(http.StatusOK, "msg_demo.html", gin.H{
		"title": "消息测试",
	})
}

// 消息推送
func msgPush(c *gin.Context) {
	data, err := _getNodes()
	if err != nil {
		c.HTML(http.StatusBadGateway, "error.html", gin.H{
			"error": err,
		})
		return
	}

	c.HTML(http.StatusOK, "msg_push.html", gin.H{
		"title": "消息推送",
		"data":  data,
	})
}

// 消息消费
func msgPop(c *gin.Context) {
	nodes, err := _getNodes()
	if err != nil {
		c.HTML(http.StatusBadGateway, "error.html", gin.H{
			"error": err,
		})
		return
	}

	topics, err := _getTopics()
	if err != nil {
		c.HTML(http.StatusBadGateway, "error.html", gin.H{
			"error": err,
		})
		return
	}

	c.HTML(http.StatusOK, "msg_pop.html", gin.H{
		"title":  "消息消费",
		"nodes":  nodes,
		"topics": topics,
	})
}

// 消息确认
func msgAck(c *gin.Context) {
	nodes, err := _getNodes()
	if err != nil {
		c.HTML(http.StatusBadGateway, "error.html", gin.H{
			"error": err,
		})
		return
	}

	c.HTML(http.StatusOK, "msg_ack.html", gin.H{
		"title": "消息确认",
		"nodes": nodes,
	})
}

// 死信消息确认
func msgDead(c *gin.Context) {
	nodes, err := _getNodes()
	if err != nil {
		c.HTML(http.StatusBadGateway, "error.html", gin.H{
			"error": err,
		})
		return
	}

	c.HTML(http.StatusOK, "msg_dead.html", gin.H{
		"title": "消息确认",
		"nodes": nodes,
	})
}

// 推送消息
func push(c *gin.Context) {
	addr := c.PostForm("addr")
	topic := c.PostForm("topic")
	content := c.PostForm("content")
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

	m := msg{}
	m.Topic = topic
	m.Body = content
	m.Delay, _ = strconv.Atoi(delay)
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
func pop(c *gin.Context) {
	topic := c.Query("topic")
	addr := c.Query("addr")
	if len(topic) == 0 {
		c.JSON(http.StatusBadRequest, "topic is empty")
		return
	}
	if len(addr) == 0 {
		c.JSON(http.StatusBadRequest, "please select a node.")
		return
	}

	v := url.Values{}
	v.Set("topic", topic)
	gmqApi("get", "http://"+addr+"/pop", v, c)
}

// 确认消息
func ack(c *gin.Context) {
	addr := c.Query("addr")
	msgId := c.Query("msgId")
	topic := c.Query("topic")
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

	v := url.Values{}
	v.Set("topic", topic)
	v.Set("msgId", msgId)
	gmqApi("get", "http://"+addr+"/ack", v, c)
}

// 批量推送消息
func mpush(c *gin.Context) {
	c.JSON(http.StatusOK, "unsport")
}

// 设置topic信息
func set(c *gin.Context) {
	topic := c.Query("topic")
	isAutoAck := c.DefaultQuery("isAutoAck", "0")
	if len(topic) == 0 {
		c.JSON(http.StatusBadRequest, "topic is empty")
		return
	}

	v := url.Values{}
	v.Set("topic", topic)
	v.Set("isAutoAck", isAutoAck)
	gmqApi("get", API_GMQ_NODE+"/set", v, c)
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
func _getNodes() (interface{}, error) {
	resp, err := http.Get(API_GMQ_REGISTER + "/getNodes")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data respStruct
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	return data.Data, nil
}

// 获取topic
func _getTopics() (interface{}, error) {
	resp, err := http.Get(API_GMQ_NODE + "/getAllTopicStat")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data respStruct
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	return data.Data, nil
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

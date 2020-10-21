package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

var (

	// 权限
	apiKey     = ""
	secretKey  = ""
	subAccount = ""

	// 目标钉钉群
	ding = ""
	// 通知会带这个讯息，表明身份
	myName = ""

	client *FtxClient

	cfg *Config
)

type PersistData struct {
	Grids []*TradeGrid
}

type OrderMap struct {
	Orders map[string]*GridOrder
}

func NewOrderMap() *OrderMap {
	return &OrderMap{
		Orders: map[string]*GridOrder{},
	}
}

func (orderm *OrderMap) add(order *GridOrder) {
	// orderm.mutex.Lock()
	// defer orderm.mutex.Unlock()
	orderm.Orders[order.ClientId] = order
}

func (orderm *OrderMap) RangeOver(fn func(order *GridOrder) bool) {
	// orderm.mutex.Lock()
	// defer orderm.mutex.Unlock()
	for _, order := range orderm.Orders {
		if !fn(order) {
			break
		}
	}
}

func (orderm *OrderMap) remove(clientId string) {
	// orderm.mutex.Lock()
	// defer orderm.mutex.Unlock()
	delete(orderm.Orders, clientId)
}

func (orderm *OrderMap) get(clientId string) (*GridOrder, bool) {
	// orderm.mutex.Lock()
	// defer orderm.mutex.Unlock()
	order, found := orderm.Orders[clientId]
	return order, found
}

func place(clientId string, market string, side string, price float64, _type string, size float64, reduce bool, post bool) {
	log.Infoln("PlaceOrder", clientId, market, side, price, _type, size, "reduce", reduce, "postonly", post)
	if *testMode {
		return
	}

	resp, err := client.placeOrder(clientId, market, side, price, _type, size, reduce, post)
	if err != nil {
		log.Errorln("PlaceError", err)
		SendDingTalkAsync(fmt.Sprintln("发送订单失败:", market, side, price, _type, size, reduce, "原因：", err))
		return
	}
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)

	var result Result
	json.Unmarshal(b, &result)

	if result.Error != "" {
		SendDingTalkAsync(fmt.Sprintln("发送订单失败:", market, side, price, _type, size, reduce, "原因：", result.Error))
	}

	log.Infoln("PlaceResult", string(b))
}

func mustFloat(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic("invalid float " + s)
	}
	return f
}

func mustInt(s string) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		panic("invalid int " + s)
	}
	return n
}

func mustBool(s string) bool {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		panic("invalid int " + s)
	}

	return n != 0
}

func debugPositions() {
	rsp, err := client.getPositions()
	if err != nil {
		log.Println("getPositions", err)
		return
	}
	simplePrintResponse(rsp)
}

func excelBool(b bool) int {
	if b {
		return 1
	}
	return 0
}

func loadBaseConfigAndAssign(file string) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatalln("read config:", err)
	}
	var config Config
	if err := json.Unmarshal(content, &config); err != nil {
		log.Fatalln("parse config:", err)
	}

	apiKey = config.ApiKey
	secretKey = config.SecretKey
	subAccount = config.SubAccount
	myName = config.MyName
	ding = config.Ding
	cfg = &config

	client = &FtxClient{
		Client:     &http.Client{},
		Api:        apiKey,
		Secret:     []byte(secretKey),
		Subaccount: subAccount,
	}
}

type GridOrder struct {
	ClientId   string
	Id         int64
	Qty        float64
	EQty       float64
	CreateAt   time.Time
	UpdateTime time.Time
	DeleteAt   time.Time  `yaml:"-"`
	Grid       *TradeGrid `yaml:"-"`
	Side       string
}

type TradeGrid struct {
	Uuid        string
	OpenAt      float64
	CloseAt     float64
	OpenChance  float64
	CloseChance float64

	OpenTotal   float64
	CloseTotal  float64
	OpenOrders  *OrderMap
	CloseOrders *OrderMap
}

type Config struct {
	ApiKey               string          `json:"apiKey"`
	SecretKey            string          `json:"secretKey"`
	SubAccount           string          `json:"subAccount"`
	Ding                 string          `json:"ding"`
	MyName               string          `json:"myName"`
	QuickRecheckInterval int             `json:"quickRecheckInterval"`
	CheckInterval        int             `json:"checkInterval"`
	CheckMarkets         []*CheckMarkets `json:"markets"`
}

type CheckMarkets struct {
	Spot    string   `json:"spot"`
	Futures []string `json:"futures"`
}

func NewDefaultConfig() *Config {
	return &Config{
		QuickRecheckInterval: 500,
		CheckInterval:        1500,
	}
}

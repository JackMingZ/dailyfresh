package controllers

import (
	"github.com/beego/beego/v2/adapter/orm"
	orm2 "github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	beego "github.com/beego/beego/v2/server/web"
	"github.com/gomodule/redigo/redis"
	"strconv"
	"strings"
	"test/models"
	"time"
)

type OrderController struct {
	beego.Controller
}

// 展示订单
func (this *OrderController) ShowOrder() {
	GetUser(&this.Controller)
	//获取数据
	skuids := this.GetStrings("skuid")

	if len(skuids) == 0 {
		logs.Info("请求数据错误")
		this.Redirect("/user/cart", 302)
		return
	}

	//处理数据
	o := orm.NewOrm()
	conn, _ := redis.Dial("tcp", "127.0.0.1:6379")
	defer conn.Close()
	//获取用户数据
	var user models.User
	userName := this.GetSession("userName")
	user.Name = userName.(string)
	o.Read(&user, "Name")
	goodsBuffer := make([]map[string]interface{}, len(skuids))

	totalPrice := 0
	totalCount := 0
	for index, skuid := range skuids {
		temp := make(map[string]interface{})

		id, _ := strconv.Atoi(skuid)
		//查询商品数据
		var goodsSku models.GoodsSku
		goodsSku.Id = id
		o.Read(&goodsSku)
		temp["goods"] = goodsSku
		//获取商品数量
		count, _ := redis.Int(conn.Do("hget", "cart_"+strconv.Itoa(user.Id), id))
		temp["count"] = count
		amount := goodsSku.Price * count
		temp["amount"] = amount
		//计算总金额和总件数
		totalPrice += amount
		totalCount += count
		goodsBuffer[index] = temp
	}
	this.Data["goodsBuffer"] = goodsBuffer

	//传递总金额和总件数
	this.Data["totalPrice"] = totalPrice
	this.Data["totalCount"] = totalCount
	//设定运费
	transferPrice := 10
	this.Data["transferPrice"] = transferPrice

	//实付款
	this.Data["realPrice"] = totalPrice + transferPrice

	//获取地址
	var addrs []models.Address
	o.QueryTable("Address").RelatedSel("User").Filter("User__Id", user.Id).All((&addrs))
	this.Data["addrs"] = addrs
	//传递所有商品的id
	this.Data["skuids"] = skuids
	//返回视图
	this.TplName = "place_order.html"
}

// 添加订单
func (this *OrderController) AddOrder() {
	//获取数据
	addrid, _ := this.GetInt("addrid")
	payId, _ := this.GetInt("payId")
	skuid := this.GetString("skuids")
	ids := skuid[1 : len(skuid)-1]

	skuids := strings.Split(ids, " ")

	//totalPrice, _ := this.GetInt("totalPrice")
	totalCount, _ := this.GetInt("totalCount")
	transferPrice, _ := this.GetInt("transferPrice")
	realPrice, _ := this.GetInt("realPrice")
	resp := make(map[string]interface{})
	defer this.ServeJSON()
	//校验数据
	if len(skuids) == 0 {
		logs.Info("获取数据错误")
		resp["code"] = 1
		resp["errmsg"] = "数据库连接错误"
		this.Data["json"] = resp
		return
	}

	//处理数据
	o := orm.NewOrm()

	o.Begin()

	userName := this.GetSession("userName")
	var user models.User
	user.Name = userName.(string)
	err := o.Read(&user, "Name")
	if err != nil {
		logs.Info("failed to read user:", err)
		// Handle the error, possibly return
	}

	var order models.OrderInfo
	location, _ := time.LoadLocation("Asia/Shanghai") // or whatever your location is
	now := time.Now().In(location)
	order.OrderId = now.Format("20060102150405") + strconv.Itoa(user.Id)

	order.User = &user
	//order.Orderstatus=1表示未支付，2表示已支付
	order.Orderstatus = 1
	order.PayMethod = payId
	order.TotalPrice = realPrice
	order.TotalCount = totalCount
	order.TransitPrice = transferPrice
	//查询地址
	var addr models.Address
	addr.Id = addrid
	o.Read(&addr)
	order.Address = &addr

	//执行订单表插入
	_, err = o.Insert(&order)
	if err != nil {
		logs.Info(err)
	}
	conn, _ := redis.Dial("tcp", "127.0.0.1:6379")
	defer conn.Close()
	//向订单商品表插入
	for _, skuid := range skuids {
		id, _ := strconv.Atoi(skuid)

		var goods models.GoodsSku
		goods.Id = id
		i := 3
		for i > 0 {

			o.Read(&goods)
			var orderGoods models.OrderGoods
			orderGoods.GoodsSku = &goods
			orderGoods.OrderInfo = &order
			count, _ := redis.Int(conn.Do("hget", "cart_"+strconv.Itoa(user.Id), id))

			if count > goods.Stock {
				resp["code"] = 2
				resp["errmsg"] = "商品库存目前不足"
				this.Data["json"] = resp
				o.Rollback()
				return
			}
			preCount := goods.Stock
			//time.Sleep(time.Millisecond * 500)
			orderGoods.Count = count
			orderGoods.Price = count * goods.Price
			_, err := o.Insert(&orderGoods)
			if err != nil {
				logs.Info(err)
			}

			goods.Stock -= count
			goods.Sales += count
			UpdateCount, _ := o.QueryTable("GoodsSku").Filter("Id", goods.Id).
				Filter("Stock", preCount).
				Update(orm2.Params(orm.Params{"Stock": goods.Stock, "Sales": goods.Sales}))
			if UpdateCount == 0 {
				if i > 0 {
					i -= 1
					continue
				}
				resp["code"] = 3
				resp["errmsg"] = "商品库存改变，提交订单失败"
				this.Data["json"] = resp
				o.Rollback()
				return
			} else {
				conn.Do("hdel", "cart_"+strconv.Itoa(user.Id), goods.Id)
				break
			}
		}
	}

	//返回数据
	o.Commit()
	resp["code"] = 5
	resp["errmsg"] = "ok"
	this.Data["json"] = resp
}

/*// 处理支付
func (this *OrderController) HandlePay() {
	var publicKey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA2Fb+ynB32f4Rpa9/kV+Q\nJSPaP8qmlabzryKwBiRgXCLuYSZ3cBBByMQThk/eiHPsqPO4QHqie8NRc81oXfl0\nIQFQrqPR1sOdWcz4853SZWti/nONz8IrB217dFkI3jj3NZBCzJdekos82/dSKPui\n9ceTgHK6lQ5BkRyvdAnM/rLVXtMlEOg0vwnbUN10IOyxsVwx6A1J+KFGyiboNgdO\nVfln33PmcTqQv1PbxwXmXrAcAgihDpHWAa9ntaKjVOe43zRE2hXauUvl4sFTF1HN\nVQvo4h5xfd0AGXv4apYSe1Qqna0xlA1FOT1KgbloWx5/+VI7jZLO9TRi5dfPYSZQ\nTwIDAQAB"
	var appId = "2088721004957722"
	var privateKey = "MIIEvwIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQDYVv7KcHfZ/hGl\nr3+RX5AlI9o/yqaVpvOvIrAGJGBcIu5hJndwEEHIxBOGT96Ic+yo87hAeqJ7w1Fz\nzWhd+XQhAVCuo9HWw51ZzPjzndJla2L+c43PwisHbXt0WQjeOPc1kELMl16Sizzb\n91Io+6L1x5OAcrqVDkGRHK90Ccz+stVe0yUQ6DS/CdtQ3XQg7LGxXDHoDUn4oUbK\nJug2B05V+Wffc+ZxOpC/U9vHBeZesBwCCKEOkdYBr2e1oqNU57jfNETaFdq5S+Xi\nwVMXUc1VC+jiHnF93QAZe/hqlhJ7VCqdrTGUDUU5PUqBuWhbHn/5UjuNks71NGLl\n189hJlBPAgMBAAECggEATFVFyYAticlPyLpHtK+XWMNxuphydtNVoDIJEeG77kaU\n/cpo0i2qSICGsxlzV4ovst1r4bRjqG+eSdHsRVxDUXH5WeWLoM+csZbVMIA7QHXH\nlCiJnQjRzekfakoQCvjmoQupxi6Su/pNGwAFCVjggwBMV0Ij+3vwPpd0gOkEX8lr\nqKOGWzsV1L8oLa9drmpQ4VMMgBq4pyfHaH7KRQpRhzd5UVEeqrCoo7i34eHfU+15\nMqj/Nt20Gb2IhPash45qnG/xm3BIbtUTxPNCDoTtFcSs5Ya5y8bCdncz98Q+/fY/\nNd0iE9FHI/VvX3q6ZSt15klziF936jDqB4mAzRMBIQKBgQDg00ZosSHmCDOnpO4C\nHYVNIHiFfHou4nEw6fQKYgwDIHaVUFATD/Hakb0e0JziM4cJWi2KMe96AvID+E/r\nnPJ/Hw9cf0M/cDa+FX0y1xUy+E3018IHQm7stYzu5Oy5vug2c5dclpy7tHGxk/D2\nNvhW1sxfJR3eIN6Jv7lE4X7o8wKBgQD2VoH8dyPdngb90G34CcfZWB6MuRQYAqd9\nAgGuiSMIDxtIboVK8jrQbwYolfFC/Hov1jIhARGnakj9JwkaT+eP+3SBfgG5oQ5l\nSdhO0SRXHGEbS4y8N3A4h72AekdGBi72jfCL3kWBcg6mNCAa/k+TWbTp5OcwDBQV\n5nIOFEUSNQKBgQCtAegsqCJt4eHeIA0Hk7AAqfwUvLVJXve7rE0fsFOOFG0seaEl\nCiATEhN2oxIW/4/qonpo3gRq39ldNLhLl3sEV+J6S3R0XOXDYMX3WYv2rR1QTLgC\n3hx+CzdonsGMLlyDim/vz/bMew8Cl9XVond4W9LpZKaXSLP3TJJFb0E6AQKBgQDx\nvqWj2FvHMj0UOsagwyBwCA069qpkgb5SbHSwDv7k+sZAh82hZiQXxszZaYSxw0o5\nxc++Gel0TVbBsNw7CS1rXE7SgZE51XdmKVjwyEgMgNo/Sh4b25/yqitreRSXAJx3\n84WcDY5SYVdE/iR/uRDovwFPBAdpXIEdmOBXNsct/QKBgQDM02VdRoRMyZzXBemk\n5HAoXLPdHhoAmHZ1us2usMyQxK00gqy/DTdpZbu40Gg0thzGZh4MWBRiSHDbkOn6\nutk6ZGmIE4wNMztommePkgpIDXL4/AEJGZCBKl1UPQVx4F0DYwuuVXt2WhI5q9Gy\nkylhqTvPhSan541RAB3zEpmDnQ=="
	client := alipay.New(appId, publicKey, privateKey, false)

	var p = alipay.AliPayTradePagePay{}
	p.NotifyURL = "http://xxx"
	p.ReturnURL = "http://127.0.0.1:8080/user/userCenterOrder"
	p.Subject = "天天生鲜购物平台"
	p.OutTradeNo = "202307140923"
	p.TotalAmount = "10000.00"
	p.ProductCode = "FAST_INSTANT_TRADE_PAY"

	url, err := client.TradePagePay(p)
	if err != nil {
		logs.Info("Error in TradePagePay:++++++++++++++++++", err)
		return
	}

	if url == nil {
		logs.Info("Received nil url from TradePagePay++++++++++++++++", url)
		return
	}

	// 这个 payURL 即是用于打开支付宝支付页面的 URL，可将输出的内容复制，到浏览器中访问该 URL 即可打开支付页面。
	var payURL = url.String()
	logs.Info("+++++++++++++++++++++++++++++", payURL)
	this.Redirect(payURL, 302)
}*/

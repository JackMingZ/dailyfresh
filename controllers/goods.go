package controllers

import (
	"fmt"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	beego "github.com/beego/beego/v2/server/web"
	"github.com/gomodule/redigo/redis"
	"strconv"
	"test/models"
)

type GoodsController struct {
	beego.Controller
}

func GetUser(this *beego.Controller) interface{} {
	userName := this.GetSession("userName")
	if userName == nil {
		this.Data["userName"] = ""
	} else {
		this.Data["userName"] = userName.(string)
	}
	return userName
}

func (this *GoodsController) ShowIndex() {
	GetUser(&this.Controller)
	o := orm.NewOrm()
	//获取类型数据
	var goodsTypes []models.GoodsType
	o.QueryTable("GoodsType").All(&goodsTypes)
	this.Data["goodsTypes"] = goodsTypes

	//获取轮播图数据
	var indexGoodsBanner []models.IndexGoodsBanner
	o.QueryTable("IndexGoodsBanner").OrderBy("Index").All(&indexGoodsBanner)
	this.Data["indexGoodsBanner"] = indexGoodsBanner

	//获取促销商品数据
	var promotionGoods []models.IndexPromotionBanner
	o.QueryTable("IndexPromotionBanner").OrderBy("Index").All(&promotionGoods)
	this.Data["promotionsGoods"] = promotionGoods

	//首页展示商品数据
	goods := make([]map[string]interface{}, len(goodsTypes))

	//向切片interface中插入类型数据
	for index, value := range goodsTypes {
		//获取对应类型的首页展示商品
		temp := make(map[string]interface{})
		temp["type"] = value
		goods[index] = temp
	}
	//商品数据

	for _, value := range goods {
		var textGoods []models.IndexTypeGoodsBanner
		var imgGoods []models.IndexTypeGoodsBanner
		//获取文字商品数据
		o.QueryTable("IndexTypeGoodsBanner").RelatedSel("GoodsType", "GoodsSKU").OrderBy("Index").Filter("GoodsType", value["type"]).Filter("DisplayType", 0).All(&textGoods)
		//获取图片商品数据
		o.QueryTable("IndexTypeGoodsBanner").RelatedSel("GoodsType", "GoodsSKU").OrderBy("Index").Filter("GoodsType", value["type"]).Filter("DisplayType", 1).All(&imgGoods)

		value["textGoods"] = textGoods
		value["imgGoods"] = imgGoods
	}
	this.Data["goods"] = goods

	this.TplName = "index.html"
}
func ShowLaout(this *beego.Controller) {
	//查询类型
	o := orm.NewOrm()
	var types []models.GoodsType
	o.QueryTable("GoodsType").All(&types)
	this.Data["types"] = types
	//获取用户信息
	GetUser(this)
	//指定layout
	this.Layout = "goodsLayout.html"
}

func (this *GoodsController) ShowGoodsDetail() {
	//beego.SetStaticPath("/images", "D:\\Goland_\\goproject\\src\\github.com\\Jack_Ming\\test\\test\\static\\images")

	//获取数据
	id, err := this.GetInt("id")
	//校验数据
	if err != nil {
		logs.Info("浏览器请求错误")
		logs.Info(err)
		this.Redirect("/", 302)
		return
	}
	//处理数据
	//处理数据
	o := orm.NewOrm()
	var goodsSku models.GoodsSku
	goodsSku.Id = id

	o.QueryTable("GoodsSku").RelatedSel("GoodsType", "Goods").Filter("Id", id).One(&goodsSku)
	//获取同类型时间考前的两条商品数据
	var goodsNew []models.GoodsSku
	o.QueryTable("GoodsSku").RelatedSel("GoodsType").Filter("GoodsType", goodsSku.GoodsType).OrderBy("Time").Limit(2, 0).All(&goodsNew)

	this.Data["goodsNew"] = goodsNew

	//返回视图
	fmt.Println(goodsSku)
	this.Data["goodsSku"] = goodsSku

	//添加历史浏览记录
	//判断用户登录
	userName := this.GetSession("userName")
	if userName != nil {
		//查询用户信息
		o := orm.NewOrm()
		var user models.User
		user.Name = userName.(string)
		o.Read(&user, "Name")
		// 添加历史记录,用redis存储
		conn, err := redis.Dial("tcp", "127.0.0.1:6379", redis.DialPassword(""))

		defer conn.Close()
		if err != nil {
			logs.Info("redis连接错误", err)
		}
		//把以前相同商品的历史浏览记录删除
		conn.Do("lrem", "history_"+strconv.Itoa(user.Id), 0, id)
		//添加新的商品浏览记录
		conn.Do("lpush", "history_"+strconv.Itoa(user.Id), id)
	}

	ShowLaout(&this.Controller)
	cartCount := GetCartCount(&this.Controller)
	this.Data["cartCount"] = cartCount
	this.TplName = "detail.html"
}

func (this *GoodsController) ShowList() {
	//获取数据
	id, err := this.GetInt("typeId")
	//校验数据
	if err != nil {
		logs.Info("请求路径错误")
		this.Redirect("/", 302)
		return
	}
	//处理数据
	ShowLaout(&this.Controller)
	//获取新品
	o := orm.NewOrm()
	var goodsNew []models.GoodsSku
	o.QueryTable("GoodsSku").RelatedSel("GoodsType").Filter("GoodsType__Id", id).OrderBy("Time").Limit(3, 0).All(&goodsNew)
	this.Data["goodsNew"] = goodsNew

	//获取商品
	var goods []models.GoodsSku
	o.QueryTable("GoodsSku").RelatedSel("GoodsType").Filter("GoodsType__Id", id).All(&goods)
	this.Data["goods"] = goods

	//返回视图
	this.TplName = "list.html"
}

// 处理搜索
func (this *GoodsController) HandleSearch() {
	//获取数据
	goodsName := this.GetString("goodsName")

	o := orm.NewOrm()
	var goods []models.GoodsSku
	//校验数据
	if goodsName == "" {
		o.QueryTable("GoodsSku").All(&goods)
		this.Data["goods"] = goods
		ShowLaout(&this.Controller)
		this.TplName = "search.html"
		return
	}
	//处理数据
	o.QueryTable("GoodsSku").Filter("Name__icontains", goodsName).All(&goods)
	//返回视图
	this.Data["goods"] = goods
	ShowLaout(&this.Controller)
	this.TplName = "search.html"
}

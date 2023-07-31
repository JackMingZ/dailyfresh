package routers

import (
	beego "github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
	"test/controllers"
)

func init() {
	beego.InsertFilter("/user/*", beego.BeforeExec, filterFunc)
	//beego.Router("/", &controllers.MainController{})
	beego.Router("/index", &controllers.MainController{})
	beego.Router("/register", &controllers.UserController{}, "get:ShowReg;post:HandleReg")
	beego.Router("/active", &controllers.UserController{}, "get:ActiveUser")
	beego.Router("/login", &controllers.UserController{}, "get:ShowLogin;post:HandleLogin")
	//跳转首页
	beego.Router("/", &controllers.GoodsController{}, "get:ShowIndex")
	//退出登录
	beego.Router("/user/logout", &controllers.UserController{}, "get:Logout")
	//用户中心信息页
	beego.Router("/user/userCenterInfo", &controllers.UserController{}, "get:ShowUserCenterInfo")
	//用户中心订单页
	beego.Router("/user/userCenterOrder", &controllers.UserController{}, "get:ShowUserCenterOrder")
	//用户中心地址页
	beego.Router("/user/userCenterSite", &controllers.UserController{}, "get:ShowUserCenterSite;post:HandleUserCenterSite")
	//商品详情展示
	beego.Router("/goodsDetail", &controllers.GoodsController{}, "get:ShowGoodsDetail")
	//获取商品列表页
	beego.Router("/goodsList", &controllers.GoodsController{}, "get:ShowList")
	//商品搜索
	beego.Router("/goodsSearch", &controllers.GoodsController{}, "post:HandleSearch")
	//添加购物车
	beego.Router("/user/addCart", &controllers.CartController{}, "post:HandleAddCart")
	//展示购物车页面
	beego.Router("/user/cart", &controllers.CartController{}, "get:ShowCart")
	//更新购物车数量
	beego.Router("/user/UpdateCart", &controllers.CartController{}, "post:HandleUpdateCart")
	//删除购物车数据
	beego.Router("/user/deleteCart", &controllers.CartController{}, "post:DeleteCart")
	//展示订单页面
	beego.Router("/user/showOrder", &controllers.OrderController{}, "post:ShowOrder")
	//添加订单
	beego.Router("/user/addOrder", &controllers.OrderController{}, "post:AddOrder")
	//beego.Router("/user/pay", &controllers.OrderController{}, "get:HandlePay")
}

var filterFunc = func(ctx *context.Context) {
	userName := ctx.Input.Session("userName")
	if userName == nil {
		ctx.Redirect(302, "/login")
		return
	}
}

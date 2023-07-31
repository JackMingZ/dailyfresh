package controllers

import (
	"encoding/base64"
	"github.com/beego/beego/v2/adapter/orm"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/core/utils"
	beego "github.com/beego/beego/v2/server/web"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gomodule/redigo/redis"
	"regexp"
	"strconv"
	"test/models"
)

type UserController struct {
	beego.Controller
}

// 显示注册页面
func (this *UserController) ShowReg() {
	this.TplName = "register.html"
}

// 处理注册页面
func (this *UserController) HandleReg() {
	//1.获取数据
	userName := this.GetString("user_name")
	pwd := this.GetString("pwd")
	cpwd := this.GetString("cpwd")
	email := this.GetString("email")
	//2.校验数据
	if userName == "" || pwd == "" || cpwd == "" || email == "" {
		this.Data["errmsg"] = "信息不完整，请重新注册"
		this.TplName = "register.html"
		return
	}
	if pwd != cpwd {
		this.Data["errmsg"] = "两次输入的密码不一致，请重新输入"
		this.TplName = "register.html"
		return
	}
	//如果email满足正则表达式定义的格式（也就是说，email是一个有效的邮箱地址），
	//那么FindString方法将返回email本身。
	//，FindString方法将返回一个空字符串。这个结果将被存储在res变量中。
	reg, _ := regexp.Compile("^[A-Za-z0-9\u4e00-\u9fa5]+@[a-zA-Z0-9_-]+(\\.[a-zA-Z0-9_-]+)+$")
	res := reg.FindString(email)
	if res == "" {
		this.Data["errmsg"] = "邮箱格式不正确"
		this.TplName = "register.html"
		return
	}
	//3.校验数据
	o := orm.NewOrm()
	var user models.User
	user.Name = userName
	user.Password = pwd
	user.Email = email
	_, err := o.Insert(&user)
	if err != nil {
		this.Data["errmsg"] = "注册失败,请更换数据注册"
		this.TplName = "register.html"
		return
	}
	//logs.Info(o)
	//发送邮件，激活
	emailConfig := `{"username":"1437931761@qq.com","password":"mnypfuwsphhvffah","host":"smtp.qq.com","port":587}`
	//logs.Info(emailConfig)
	emailConn := utils.NewEMail(emailConfig)
	emailConn.From = "1437931761@qq.com"
	emailConn.To = []string{email}
	emailConn.Subject = "用户注册"
	//发送的是激活地址
	emailConn.Text = "请在浏览器中输入：192.168.161.21:8080/active?id=" + strconv.Itoa(user.Id) + "\n若不输入，将无法激活账户"
	emailConn.Send()
	//4.返回视图
	this.Ctx.WriteString("注册成功，请激活用户！")
}

// 激活处理
func (this *UserController) ActiveUser() {
	//获取数据
	id, err := this.GetInt("id")
	//校验数据
	if err != nil {
		this.Data["errmsg"] = "要激活的用户不存在"
		logs.Info(err)
		this.TplName = "register.html"
		return
	}
	//处理数据
	//更新对象
	o := orm.NewOrm()
	var user models.User
	user.Id = id
	err = o.Read(&user)
	if err != nil {
		this.Data["errmsg"] = "要激活的用户不存在"
		logs.Info(err)
		this.TplName = "register.html"
		return
	}
	user.Active = true
	o.Update(&user)
	//返回视图
	this.Redirect("/login", 302)
}

// 展示登录页面
func (this *UserController) ShowLogin() {
	userName := this.Ctx.GetCookie("userName")
	temp, _ := base64.StdEncoding.DecodeString(userName)
	if string(temp) == "" {
		this.Data["userName"] = ""
		this.Data["checked"] = ""
	} else {
		this.Data["userName"] = string(temp)
		this.Data["checked"] = "checked"
	}
	this.TplName = "login.html"
}

// 处理登录业务
func (this *UserController) HandleLogin() {
	//1.获取数据
	userName := this.GetString("username")
	pwd := this.GetString("pwd")
	//2.校验数据
	if userName == "" || pwd == "" {
		this.Data["errmsg"] = "登录数据不完整"
		this.TplName = "login.html"
		return
	}
	//3.处理数据
	o := orm.NewOrm()
	var user models.User
	user.Name = userName
	err := o.Read(&user, "Name")
	if err != nil {
		this.Data["errmsg"] = "用户名或密码错误"
		this.TplName = "login.html"
		return
	}
	if user.Password != pwd {
		this.Data["errmsg"] = "用户名或密码错误"
		this.TplName = "login.html"
		return
	}
	if user.Active != true {
		this.Data["errmsg"] = "用户未激活，请激活后登录"
		this.TplName = "login.html"
		return
	}
	//4.返回视图
	remember := this.GetString("remember")
	//base64加密
	if remember == "on" {

		temp := base64.StdEncoding.EncodeToString([]byte(userName))
		this.Ctx.SetCookie("userName", temp, 24*3600*30)
	} else {
		this.Ctx.SetCookie("userName", userName, -1)
	}
	this.SetSession("userName", userName)
	this.Redirect("/", 302)
	//this.Ctx.WriteString("登录成功")
}

// 退出登录
func (this *UserController) Logout() {
	//删除Session
	this.DelSession("userName")
	//跳转视图
	this.Redirect("/login.html", 302)
}

// 展示用户中心信息
func (this *UserController) ShowUserCenterInfo() {
	userName := GetUser(&this.Controller).(string)
	this.Data["userName"] = userName
	//查询地址表内容
	o := orm.NewOrm()
	//高级查询  做表关联
	var addr models.Address
	err := o.QueryTable("Address").RelatedSel("User").Filter("User__Name", userName).Filter("Isdefault", true).One(&addr)
	if err != nil {
		logs.Info("打印错误========", err)
	}
	//this.Data["addr"]=addr
	if addr.Id == 0 {
		this.Data["addr"] = ""
	} else {
		this.Data["addr"] = addr
	}

	//获取历史浏览记录
	conn, err := redis.Dial("tcp", "127.0.0.1:6379")
	defer conn.Close()
	if err != nil {
		logs.Info("连接错误", err)
	}
	//获取用户ID
	var user models.User
	user.Name = userName
	o.Read(&user, "Name")

	rep, err := conn.Do("lrange", "history_"+strconv.Itoa(user.Id), 0, 4)

	if err != nil {
		logs.Info("err::::::::::", err)
	}
	//回复助手函数
	goodsIDs, err2 := redis.Ints(rep, err)
	if err2 != nil {
		logs.Info("打印错误========err2\n", err2)
	}
	//logs.Info("打印goodsIDs", goodsIDs)

	var goodsSkus []models.GoodsSku

	for _, value := range goodsIDs {
		var goods models.GoodsSku
		goods.Id = value
		o.Read(&goods)
		goodsSkus = append(goodsSkus, goods)
	}
	//logs.Info("打印goodsSkus========", goodsSkus)
	this.Data["goodsSkus"] = goodsSkus
	this.Layout = "userCenterLayout.html"
	this.TplName = "user_center_info.html"
}

// 展示用户中心订单页
func (this *UserController) ShowUserCenterOrder() {
	//获取订单表的数据
	o := orm.NewOrm()
	userName := GetUser(&this.Controller).(string)

	var user models.User
	user.Name = userName
	o.Read(&user, "Name")
	var orderInfos []models.OrderInfo
	o.QueryTable("OrderInfo").Filter("User__Id", user.Id).All(&orderInfos)

	goodsBuffer := make([]map[string]interface{}, len(orderInfos))

	for index, orderInfo := range orderInfos {

		var orderGoods []models.OrderGoods
		o.QueryTable("OrderGoods").RelatedSel("OrderInfo", "GoodsSku").Filter("OrderInfo__Id", orderInfo.Id).All(&orderGoods)

		temp := make(map[string]interface{})
		temp["orderInfo"] = orderInfo
		temp["orderGoods"] = orderGoods

		goodsBuffer[index] = temp

	}

	this.Data["goodsBuffer"] = goodsBuffer
	this.Layout = "userCenterLayout.html"
	this.TplName = "user_center_order.html"

}

// 展示用户中心地址页
func (this *UserController) ShowUserCenterSite() {
	userName := GetUser(&this.Controller).(string)
	//this.Data["userName"] = userName

	//获取地址信息
	o := orm.NewOrm()
	var addr models.Address
	o.QueryTable("Address").RelatedSel("User").Filter("User__Name", userName).Filter("Isdefault", true).One(&addr)

	//传递给视图
	this.Data["addr"] = addr
	this.Layout = "userCenterLayout.html"
	this.TplName = "user_center_site.html"

}

// 处理用户中心地址数据
func (this *UserController) HandleUserCenterSite() {
	//获取数据
	receiver := this.GetString("receiver")
	addr := this.GetString("addr")
	zipCode := this.GetString("zipCode")
	phone := this.GetString("phone")
	//校验数据
	if receiver == "" || addr == "" || zipCode == "" || phone == "" {
		logs.Info("添加数据不完整")
		this.Redirect("/user/userCenterSite", 302)
		return
	}
	//处理数据
	//插入操作
	o := orm.NewOrm()
	var addrUser models.Address
	addrUser.Isdefault = true
	err := o.Read(&addrUser, "Isdefault")
	//添加默认地址前把原来设置的默认地址更新成非默认地址
	if err == nil {
		addrUser.Isdefault = false
		o.Update(&addrUser)
	}

	//更新默认地址时，给原来的地址对象的ID赋值了，
	//这时用原来的地址对象插入，用原来的ID做插入操作会报错
	//关联
	userName := this.GetSession("userName")
	var user models.User
	user.Name = userName.(string)
	o.Read(&user, "Name")
	var addUserNew models.Address
	addUserNew.Receiver = receiver
	addUserNew.Addr = addr
	addUserNew.Zipcode = zipCode
	addUserNew.Phone = phone
	addUserNew.Isdefault = true
	addUserNew.User = &user
	o.Insert(&addUserNew)
	//o.InsertOrUpdate(&addrUser)

	//返回视图
	this.Redirect("/user/userCenterSite", 302)
}

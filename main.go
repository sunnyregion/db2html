package main //package名称，主运行的package名规定为main
//包引入区
import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	//	"time"
	"flag"

	"github.com/astaxie/beego/orm"
	"github.com/coopernurse/gorp"
	_ "github.com/go-sql-driver/mysql"
	"github.com/modood/table"
	"github.com/sunnyregion/color"
	"github.com/sunnyregion/sunnyini"
)

var (
	dbhostsip  = "127.0.0.1" //IP地址
	port       = "3306"
	dbusername = "dfdba"   //用户名
	dbpassword = "Dt1210k" //密码
	dbname     = "dface"   //表名

	db        *sql.DB
	bFlag     = flag.Bool("b", false, "是否有数据库配置文件,默认配置文件是config.ini。")
	pFlag     = flag.Bool("p", false, "是否打印出显示信息。")
	mFlag     = flag.Bool("m", false, "是否保存为MarkDown文件。")
	oFileName = flag.String("o", "README.html", "输出的文件名。")
)

// 判断有没有错误
func checkErr(err error) {
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
}

//ResultStruct 返回结果struct
type ResultStruct struct {
	Field   string //字段名
	Type    string //类型
	Null    string //可否为空
	Key     string //是否主键
	Default string //默认值是什么
	Extra   string //额外
	Comment string //备注
}

//TablesList 表名
type TablesList struct {
	Name string
}

// 初始化
func init() {
	//此处使用ini文件调入
	flag.Parse()
	if *bFlag {
		getIni()
	} else {
		fmt.Println("🎉🎉🎉👍💁👌========>>", `没有数据库参数`, "<<=======⚽🎍😍🎉🎉🎉")
		os.Exit(0)
	}
	var err error
	strSQL := dbusername + `:` + dbpassword + `@tcp(` + dbhostsip + `:` + port + `)/` + dbname + `?charset=utf8`
	db, err = sql.Open("mysql", strSQL)
	//	fmt.Println(strSQL)
	checkErr(err)

	orm.RegisterDriver("mysql", orm.DRMySQL)

	orm.RegisterDataBase("default", "mysql", strSQL)

	//	fmt.Printf("数据库连接成功！%s\n", strSQL)
}

//getIni  取得ini文件mysql的配置
func getIni() {
	f := sunnyini.NewIniFile()
	f.Readfile("config.ini")
	// 获取所有的Section
	_ = f.GetSection()
	// 获取某一个section下的键值对
	describ, v := f.GetValue("mysql")
	if describ == "" { // 有数据
		dbhostsip = v[0]["dbhostsip"] //IP地址
		port = v[1]["port"]
		dbusername = v[2]["dbusername"] //用户名
		dbpassword = v[3]["dbpassword"] //密码
		dbname = v[4]["dbname"]         //表名
	} else {
		fmt.Println(describ)
	}
}

//GetTablesName 取得表名列表
func GetTablesName() []string {

	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{"InnoDB", "UTF8"}}
	tables := []string{}

	dbmap.Select(&tables, "SHOW TABLES;")

	tArray := []TablesList{}
	var tl TablesList
	for _, tname := range tables {

		tl.Name = tname
		tArray = append(tArray, tl)
	}
	table.OutputA(tArray)
	return tables
}

//getFieldsInfo 使用单线程的方式读取表信息
func getFieldsInfo(tables []string) {
	o := orm.NewOrm()
	var strTmp string
	for _, tname := range tables {
		var results []orm.Params
		var rsStruct ResultStruct
		_, err := o.Raw(fmt.Sprintf("show create table %s", tname)).Values(&results)
		if err != nil {
			fmt.Println(err)
		}
		strTmp = results[0][`Create Table`].(string)

		comIndex := strings.Index(strTmp, `COMMENT=`)
		c := color.New(color.FgMagenta).Add(color.Underline)
		if *pFlag {
			fmt.Println("")
			if comIndex != -1 {
				c.Println("-------------表名:", tname, strTmp[comIndex+9:len(strTmp)-1], "-----------")
			} else {
				c.Println("-------------表名:", tname, "-----------")
			}
			c = color.New(color.FgCyan).Add(color.Underline)
			c.Println(strTmp)
			fmt.Println("")
		}

		strTmp = strTmp[strings.Index(strTmp, `(`)+1 : strings.LastIndexAny(strTmp, `)`)]
		arrFields := strings.Split(strTmp, `,`)

		_, err = o.Raw(fmt.Sprintf("describe %s", tname)).Values(&results)
		tStruct := []ResultStruct{}
		for _, field := range results {
			stm := field["Field"].(string)
			rsStruct.Field = stm
			for _, af := range arrFields {
				i := strings.Index(af, `COMMENT '`)
				if strings.Index(af, stm) > 0 && i > 0 {
					rsStruct.Comment = af[i+len(`COMMENT '`) : len(af)-1]
				}
			}
			switch field["Default"].(type) {
			case string:
				rsStruct.Default = field["Default"].(string)
			case int:
				rsStruct.Default = fmt.Sprint(field["Default"])
			case nil:
				rsStruct.Default = `--`
			default:
				rsStruct.Default = `--`
			}
			rsStruct.Null = field["Null"].(string)
			stm = field["Extra"].(string)
			if len(stm) > 0 {
				rsStruct.Extra = stm
			} else {
				rsStruct.Extra = `--`
			}

			rsStruct.Type = field["Type"].(string)
			stm = field["Key"].(string)
			if len(stm) > 0 {
				rsStruct.Key = stm
			} else {
				rsStruct.Key = `--`
			}

			tStruct = append(tStruct, rsStruct)
		}
		if *pFlag {
			table.OutputA(tStruct)
		}
		if *mFlag {

			for k, v := range tStruct {
				fmt.Printf("k:=%v===v:%v\r\n", k, v)
			}
		}
	}
}

//getFieldInfoByChan 使用多线程
func getFieldInfoByChan(ch chan string) {
	o := orm.NewOrm()
	var (
		strTmp string
		strSQL string
	)
	tname := <-ch
	var results []orm.Params
	var rsStruct ResultStruct
	_, err := o.Raw(fmt.Sprintf("show create table %s", tname)).Values(&results)
	if err != nil {
		fmt.Println(err)
	}
	strTmp = results[0][`Create Table`].(string)
	strSQL = strTmp

	strTmp = strTmp[strings.Index(strTmp, `(`)+1 : strings.LastIndexAny(strTmp, `)`)]
	arrFields := strings.Split(strTmp, `,`)
	fmt.Println("")

	_, err = o.Raw(fmt.Sprintf("describe %s", tname)).Values(&results)
	tStruct := []ResultStruct{}
	for _, field := range results {
		stm := field["Field"].(string)
		rsStruct.Field = stm
		for _, af := range arrFields {
			i := strings.Index(af, `COMMENT '`)

			if strings.Index(af, stm) > 0 && i > 0 {
				rsStruct.Comment = af[i+len(`COMMENT '`) : len(af)-1]
			}
		}
		switch field["Default"].(type) {
		case string:
			rsStruct.Default = field["Default"].(string)
		case int:
			rsStruct.Default = fmt.Sprint(field["Default"])
		case nil:
			rsStruct.Default = `--`
		default:
			rsStruct.Default = `--`
		}
		rsStruct.Null = field["Null"].(string)
		stm = field["Extra"].(string)
		if len(stm) > 0 {
			rsStruct.Extra = stm
		} else {
			rsStruct.Extra = `--`
		}

		rsStruct.Type = field["Type"].(string)
		stm = field["Key"].(string)
		if len(stm) > 0 {
			rsStruct.Key = stm
		} else {
			rsStruct.Key = `--`
		}

		tStruct = append(tStruct, rsStruct)
	}
	table.OutputA(tStruct)
	comIndex := strings.Index(strSQL, `COMMENT=`)
	if comIndex != -1 {
		fmt.Println("-------------表名:", tname, strSQL[comIndex+9:len(strSQL)-1], "-----------")
	} else {
		fmt.Println("-------------表名:", tname, "-----------")
	}
	fmt.Println(strSQL)

}

//主函数
func main() {
	//c := color.New(color.FgCyan).Add(color.Underline)
	fmt.Println(`
	_____  ____    ___    _    _ _______ __  __ _      
	|  __ \|  _ \  |__ \  | |  | |__   __|  \/  | |     
	| |  | | |_) |    ) | | |__| |  | |  | \  / | |     
	| |  | |  _ <    / /  |  __  |  | |  | |\/| | |     
	| |__| | |_) |  / /_  | |  | |  | |  | |  | | |____ 
	|_____/|____/  |____| |_|  |_|  |_|  |_|  |_|______
`)
	tables := GetTablesName()
	fmt.Println(tables)
	// if *mFlag {
	// 	fmt.Println("🎉🎉🎉👍💁👌========>>", `输出MarkDown`, "<<=======⚽🎍😍🎉🎉🎉")
	// } else {
	getFieldsInfo(tables)
	// }

	// 下面是多线程的方法
	//	ch := make(chan string, len(tables))
	//	for _, tname := range tables {
	//		ch <- tname
	//		go getFieldInfoByChan(ch)
	//	}
	//	time.Sleep(4 * 1e9)
}

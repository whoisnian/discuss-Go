package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var PORT = "8080"       //监听端口
var dbname = "discuss"  //数据库名称
var dbuser = "username" //数据库用户名
var dbpass = "password" //数据库密码

var db *sql.DB
var templates *template.Template

//加载模板
func initTemplates() {
	templates = template.New("")
	templates = templates.Funcs(template.FuncMap{"add": Add})
	templates = templates.Funcs(template.FuncMap{"pages": Pages})
	templates = templates.Funcs(template.FuncMap{"br": Br})
	templates = template.Must(templates.ParseFiles(
		"views/include/header.tmpl",
		"views/include/footer.tmpl",
		"views/index.tmpl",
		"views/signup.tmpl",
		"views/login.tmpl",
		"views/404.tmpl"))
}

//计算页码
func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func Add(a, b int) int {
	return a + b
}

func Pages(PageNum, SumNum int) []int {
	var res []int
	l := Max(1, PageNum-5)
	l = Min(l, SumNum-9)
	r := Min(l+9, SumNum)
	for i := l; i <= r; i++ {
		res = append(res, i)
	}
	return res
}

//换行符
func Br(Content string) []string {
	return strings.Split(Content, "\n")
}

//页面结构
type Message struct {
	Username string
	Content  string
	Time     string
}
type IndexPage struct {
	EditMessage string
	Error       string
	Username    string
	Messages    []Message
	PageNum     int
	SumNum      int
}

type SignupAndLoginPage struct {
	Username string
	Error    string
}

//md5
func md5en(s string) string {
	res := md5.Sum([]byte(s))
	return hex.EncodeToString(res[:])
}

//base64
func base64en(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}
func base64de(s string) (string, error) {
	res, err := base64.StdEncoding.DecodeString(s)
	return string(res), err
}

//主页
func indexHandler(w http.ResponseWriter, r *http.Request) {
	indexPage := &IndexPage{}
	cookie, err := r.Cookie("logged")
	var username string
	if err == nil && cookie.Value != "" {
		username, err = base64de(cookie.Value[32:])
		if err != nil {
			username = "guest"
		} else {
			indexPage.Username = username
		}
	} else {
		username = "guest"
	}
	indexPage.PageNum, _ = strconv.Atoi(r.FormValue("p"))
	indexPage.PageNum = Max(1, indexPage.PageNum)
	err = db.QueryRow("SELECT COUNT(*) FROM message").Scan(&indexPage.SumNum)
	indexPage.SumNum = (indexPage.SumNum + 9) / 10
	if err != nil {
		fmt.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rows, err := db.Query("SELECT username, content, time FROM message WHERE messageid <= (SELECT messageid FROM message ORDER BY messageid DESC LIMIT " + strconv.Itoa(10*indexPage.PageNum-10) + ", 1) ORDER BY messageid DESC LIMIT 10")
	defer rows.Close()
	if err != nil {
		fmt.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for rows.Next() {
		var username string
		var content string
		var time string
		err = rows.Scan(&username, &content, &time)
		if err != nil {
			fmt.Println(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		indexPage.Messages = append(indexPage.Messages, Message{Username: username, Content: content, Time: time})
	}
	if r.Method == "GET" {
		err = templates.ExecuteTemplate(w, "index.tmpl", indexPage)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else if r.Method == "POST" {
		if r.FormValue("content") == "" {
			indexPage.Error = "内容不能为空"
			err = templates.ExecuteTemplate(w, "index.tmpl", indexPage)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
		if username != "guest" {
			row, err := db.Query("SELECT password from user where username=? limit 1", username)
			defer row.Close()
			if err != nil {
				fmt.Println(err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if row.Next() {
				var pass string
				err = row.Scan(&pass)
				if err != nil {
					fmt.Println(err.Error())
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if md5en(pass) != cookie.Value[:32] {
					indexPage.Error = "登录状态过期，请注销后重新登录"
				}
			} else {
				indexPage.Error = "登录状态过期，请注销后重新登录"
			}
			if indexPage.Error != "" {
				err = templates.ExecuteTemplate(w, "index.tmpl", indexPage)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				return
			}
		}
		_, err = db.Exec("INSERT message SET username=?,content=?,time=?", username, r.FormValue("content"), time.Now().Format("2006-01-02 15:04:05"))
		if err != nil {
			fmt.Println(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusFound)
	} else {
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

//注册
func signupHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("logged")
	if err == nil && cookie.Value != "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	signupandloginPage := &SignupAndLoginPage{}
	if r.Method == "GET" {
		err := templates.ExecuteTemplate(w, "signup.tmpl", signupandloginPage)
		if err != nil {
			fmt.Println(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else if r.Method == "POST" {
		signupandloginPage.Username = r.FormValue("username")
		if r.FormValue("username") == "" {
			signupandloginPage.Error = "昵称不能为空"
		} else if r.FormValue("password") == "" {
			signupandloginPage.Error = "密码不能为空"
		}
		if signupandloginPage.Error != "" {
			err := templates.ExecuteTemplate(w, "signup.tmpl", signupandloginPage)
			if err != nil {
				fmt.Println(err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}

		row, err := db.Query("SELECT 1 from user where username=? limit 1", r.FormValue("username"))
		defer row.Close()
		if err != nil {
			fmt.Println(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if row.Next() {
			signupandloginPage.Error = "该昵称已被注册"
			err = templates.ExecuteTemplate(w, "signup.tmpl", signupandloginPage)
			if err != nil {
				fmt.Println(err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
		_, err = db.Exec("INSERT user SET username=?,password=?,time=?", r.FormValue("username"), md5en(r.FormValue("password")), time.Now().Format("2006-01-02 15:04:05"))
		if err != nil {
			fmt.Println(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cookie := &http.Cookie{Name: "logged", Value: md5en(md5en(r.FormValue("password"))) + base64en(r.FormValue("username")), Path: "/"}
		http.SetCookie(w, cookie)
		http.Redirect(w, r, "/", http.StatusFound)
	} else {
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

//登录
func loginHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("logged")
	if err == nil && cookie.Value != "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	signupandloginPage := &SignupAndLoginPage{}
	if r.Method == "GET" {
		err = templates.ExecuteTemplate(w, "login.tmpl", signupandloginPage)
		if err != nil {
			fmt.Println(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else if r.Method == "POST" {
		signupandloginPage.Username = r.FormValue("username")
		if r.FormValue("username") == "" {
			signupandloginPage.Error = "昵称不能为空"
		} else if r.FormValue("password") == "" {
			signupandloginPage.Error = "密码不能为空"
		} else {
			row, err := db.Query("SELECT password from user where username=? limit 1", r.FormValue("username"))
			defer row.Close()
			if err != nil {
				fmt.Println(err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if row.Next() {
				var pass string
				err = row.Scan(&pass)
				if err != nil {
					fmt.Println(err.Error())
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if md5en(r.FormValue("password")) == pass {
					cookie := &http.Cookie{Name: "logged", Value: md5en(pass) + base64en(r.FormValue("username")), Path: "/"}
					http.SetCookie(w, cookie)
					http.Redirect(w, r, "/", http.StatusFound)
					return
				} else {
					signupandloginPage.Error = "昵称或密码错误"
				}
			} else {
				signupandloginPage.Error = "昵称或密码错误"
			}
		}
		err := templates.ExecuteTemplate(w, "login.tmpl", signupandloginPage)
		if err != nil {
			fmt.Println(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

//注销
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie := &http.Cookie{Name: "logged", Path: "/", MaxAge: -1}
	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/", http.StatusFound)
}

//路径验证
func notfoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	err := templates.ExecuteTemplate(w, "404.tmpl", nil)
	if err != nil {
		fmt.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
func makeHander(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		success := false
		if r.URL.Path == "/" {
			success = true
		} else if r.URL.Path == "/signup/" {
			success = true
		} else if r.URL.Path == "/login/" {
			success = true
		} else if r.URL.Path == "/logout/" {
			success = true
		}

		if success {
			fn(w, r)
		} else {
			notfoundHandler(w, r)
		}
	}
}

func main() {
	var err error
	db, err = sql.Open("mysql", dbuser+":"+dbpass+"@/"+dbname)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer db.Close()
	initTemplates()
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", makeHander(indexHandler))
	http.HandleFunc("/signup/", makeHander(signupHandler))
	http.HandleFunc("/login/", makeHander(loginHandler))
	http.HandleFunc("/logout/", makeHander(logoutHandler))

	fmt.Printf("Server started: <http://127.0.0.1:%v>\n", PORT)
	http.ListenAndServe(":"+PORT, nil)
}

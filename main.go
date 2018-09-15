package main

import (
	"database/sql"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	// "unicode/utf8"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/contrib/sessions"

	//"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB
var productMap sync.Map
var historyMap sync.Map

func initializeProdutMap() {
	productMap = sync.Map{}
	rows, err := db.Query("SELECT * FROM products")
	if err != nil {
		panic("Failed to select products: " + err.Error())
	}

	defer rows.Close()
	for rows.Next() {
		p := Product{}
		err = rows.Scan(&p.ID, &p.Name, &p.Description, &p.ImagePath, &p.Price, &p.CreatedAt)
		if err != nil {
			panic("Failed to scan a product: " + err.Error())
		}
		productMap.Store(p.ID, p)
	}
}

func initializeHistoryMap() {
	historyMap = sync.Map{}
	rows, err := db.Query("SELECT p.id, p.name, p.description, p.image_path, p.price, h.user_id, h.created_at " +
		"FROM histories as h " +
		"INNER JOIN products as p " +
		"ON h.product_id = p.id " +
		"ORDER BY h.id DESC")
	if err != nil {
		panic("Failed to select histories: " + err.Error())
	}

	defer rows.Close()
	for rows.Next() {
		h := History{}
		err = rows.Scan(&h.Product.ID, &h.Product.Name, &h.Product.Description, &h.Product.ImagePath, &h.Product.Price,
			&h.UserID, &h.CreatedAt)
		if err != nil {
			panic("Failed to scan history: " + err.Error())
		}
		fmt := "2006-01-02 15:04:05"
		tmp, _ := time.Parse(fmt, h.CreatedAt)
		h.Product.CreatedAt = (tmp.Add(9 * time.Hour)).Format(fmt)

		ps_, ok := historyMap.Load(h.UserID)
		var ps []Product
		if !ok {
			ps = []Product{}
		} else {
			ps = ps_.([]Product)
		}
		historyMap.Store(h.UserID, append(ps, h.Product))
	}
}

func embedIndexPage(products []PagedProductWithComments, loggedIn bool) []byte {
	var contentsBuffer []byte
	for _, p := range products {
		//if utf8.RuneCountInString(p.Description) > 70 {
		//	p.Description = string([]rune(p.Description)[:70]) + "…"
		//}
		if len(p.Description) > 210 {
			p.Description = p.Description[:210] + "…"
		}

		comments := ""
		for _, c := range p.Comments {
			//if utf8.RuneCountInString(c.Content) > 25 {
			//	c.Content = string([]rune(c.Content)[:25]) + "…"
			//}
			if len(c.Content) > 75 {
				c.Content = c.Content[:75] + "…"
			}
			comments += `<li>` + c.Content + ` by ` + c.Writer + `</li>`
		}
		pID := strconv.Itoa(p.ID)
		contentsBuffer = append(contentsBuffer, (`
			<div class="col-md-4">
				<div class="panel panel-default">
					<div class="panel-heading">
						<a href="/products/` + pID + `">` + p.Name + `</a>
					</div>
					<div class="panel-body">
						<a href="/products/` + pID + `"><img src="` + p.ImagePath + `" class="img-responsive" /></a>
						<h4>価格</h4>
						<p>` + strconv.Itoa(p.Price) + `円</p>
						<h4>商品説明</h4>
						<p>` + p.Description + `</p>
						<h4>` + strconv.Itoa(p.CommentCount) + `件のレビュー</h4>
						<ul>` + comments + `</ul>
					</div>
		`)...)
		if loggedIn {
			contentsBuffer = append(contentsBuffer, (`
				<div class="panel-footer">
					<form method="POST" action="/products/buy/` + pID + `">
						<fieldset>
							<input class="btn btn-success btn-block" type="submit" name="buy" value="購入" />
						</fieldset>
					</form>
				</div>
			`)...)
		}
		contentsBuffer = append(contentsBuffer, `</div> </div>`...)
	}
	return contentsBuffer
}
func dangerounsEnmbedMyPage(products []Product, isMe bool) []byte {
	//var contentsBuffer []byte
	contentsBuffer := make([]byte, 0, len(products)*100)
  for i, p := range products {
		if i >= 30 {
			break
		}
		if len(p.Description) > 210 {
			p.Description = p.Description[:210] + "…"
		}
		pID := strconv.Itoa(p.ID)
		contentsBuffer = append(contentsBuffer, (`
		<div class="col-md-4">
			<div class="panel panel-default">
				<div class="panel-heading">
					<a href="/products/` + pID + `">` + p.Name + `</a>
				</div>
				<div class="panel-body">
					<a href="/products/` + pID + `"><img src="` + p.ImagePath + `" class="img-responsive" /></a>
					<h4>価格</h4>
					<p>` + strconv.Itoa(p.Price) + `円</p>
					<h4>商品説明</h4>
					<p>` + p.Description + `</p>
					<h4>購入日時</h4>
					<p>` + p.CreatedAt + `</p>
				</div>
		`)...)
		if isMe {
			contentsBuffer = append(contentsBuffer, (`
				<div class="panel-footer">
					<form method="POST" action="/comments/` + pID + `">
						<fieldset>
							<div class="form-group">
								<input class="form-control" placeholder="Comment Here" name="content" value="">
							</div>
							<input class="btn btn-success btn-block" type="submit" name="send_comment" value="コメントを送信" />
						</fieldset>
					</form>
				</div>
			`)...)
		}
		contentsBuffer = append(contentsBuffer, `</div> </div>`...)
	}
	return contentsBuffer
}

func main() {
	// database setting
	user := os.Getenv("ISHOCON1_DB_USER")
	pass := os.Getenv("ISHOCON1_DB_PASSWORD")
	host := os.Getenv("ISHOCON1_DB_HOST")
	port := os.Getenv("ISHOCON1_DB_PORT")
	dbname := "ishocon1"
	db, _ = sql.Open("mysql", user+":"+pass+"@tcp("+host+":"+port+")/"+dbname)
	db.SetMaxIdleConns(5)
	gin.SetMode(gin.ReleaseMode)

	initializeHistoryMap()
	initializeProdutMap()
	r := gin.Default()
	// load templates
	r.LoadHTMLGlob("templates/*")
	//r.Use(static.Serve("/css", static.LocalFile("public/css", true)))
	//r.Use(static.Serve("/images", static.LocalFile("public/images", true)))
	//layout := "templates/layout.tmpl"

	// session store
	store := sessions.NewCookieStore([]byte("mysession"))
	store.Options(sessions.Options{HttpOnly: true})
	r.Use(sessions.Sessions("showwin_happy", store))

	// GET /login
	r.GET("/login", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Clear()
		session.Save()

		//tmpl, _ := template.ParseFiles("templates/login.tmpl")
		//r.SetHTMLTemplate(tmpl)
		c.HTML(http.StatusOK, "login", gin.H{
			"Message": "ECサイトで爆買いしよう！！！！",
		})
	})

	// POST /login
	r.POST("/login", func(c *gin.Context) {
		email := c.PostForm("email")
		pass := c.PostForm("password")

		session := sessions.Default(c)
		user, result := authenticate(email, pass)
		if result {
			// 認証成功
			session.Set("uid", user.ID)
			session.Save()

			user.UpdateLastLogin()

			c.Redirect(http.StatusSeeOther, "/")
		} else {
			// 認証失敗
			tmpl, _ := template.ParseFiles("templates/login.tmpl")
			r.SetHTMLTemplate(tmpl)
			c.HTML(http.StatusOK, "login", gin.H{
				"Message": "ログインに失敗しました",
			})
		}
	})

	// GET /logout
	r.GET("/logout", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Clear()
		session.Save()

		//tmpl, _ := template.ParseFiles("templates/login.tmpl")
		//r.SetHTMLTemplate(tmpl)
		c.Redirect(http.StatusFound, "/login")
	})

	// GET /
	r.GET("/", func(c *gin.Context) {
		cUser := currentUser(sessions.Default(c))

		page, err := strconv.Atoi(c.Query("page"))
		if err != nil {
			page = 0
		}
		products := getProductsWithCommentsAt(page)
		contentsBuffer := embedIndexPage(products, cUser.ID > 0)
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"CurrentUser": cUser,
			"Contents":    template.HTML(contentsBuffer),
		})
	})

	// GET /users/:userId
	r.GET("/users/:userId", func(c *gin.Context) {
		cUser := currentUser(sessions.Default(c))

		uid, _ := strconv.Atoi(c.Param("userId"))
		user := getUser(uid)

		products := user.BuyingHistory()

		var totalPay int
		for _, p := range products {
			totalPay += p.Price
		}
		contentsBuffer := dangerounsEnmbedMyPage(products, user.ID == cUser.ID)
		c.HTML(http.StatusOK, "mypage.tmpl", gin.H{
			"CurrentUser": cUser,
			"User":        user,
			"Contents":    template.HTML(contentsBuffer),
			"TotalPay":    totalPay,
		})
	})

	// GET /products/:productId
	r.GET("/products/:productId", func(c *gin.Context) {
		pid, _ := strconv.Atoi(c.Param("productId"))
		product := getProduct(pid)
		comments := getComments(pid)

		cUser := currentUser(sessions.Default(c))
		bought := product.isBought(cUser.ID)

		//r.SetHTMLTemplate(template.Must(template.ParseFiles(layout, "templates/product.tmpl")))
		c.HTML(http.StatusOK, "product.tmpl", gin.H{
			"CurrentUser":   cUser,
			"Product":       product,
			"Comments":      comments,
			"AlreadyBought": bought,
		})
	})

	// POST /products/buy/:productId
	r.POST("/products/buy/:productId", func(c *gin.Context) {
		// need authenticated
		if notAuthenticated(sessions.Default(c)) {
			//tmpl, _ := template.ParseFiles("templates/login.tmpl")
			//r.SetHTMLTemplate(tmpl)
			c.HTML(http.StatusForbidden, "login", gin.H{
				"Message": "先にログインをしてください",
			})
		} else {
			// buy product
			cUser := currentUser(sessions.Default(c))
			cUser.BuyProduct(c.Param("productId"))

			// redirect to user page
			//tmpl, _ := template.ParseFiles("templates/mypage.tmpl")
			//r.SetHTMLTemplate(tmpl)
			c.Redirect(http.StatusFound, "/users/"+strconv.Itoa(cUser.ID))
		}
	})

	// POST /comments/:productId
	r.POST("/comments/:productId", func(c *gin.Context) {
		// need authenticated
		if notAuthenticated(sessions.Default(c)) {
			//tmpl, _ := template.ParseFiles("templates/login.tmpl")
			//r.SetHTMLTemplate(tmpl)
			c.HTML(http.StatusForbidden, "login", gin.H{
				"Message": "先にログインをしてください",
			})
		} else {
			// create comment
			cUser := currentUser(sessions.Default(c))
			cUser.CreateComment(c.Param("productId"), c.PostForm("content"))

			// redirect to user page
			//tmpl, _ := template.ParseFiles("templates/mypage.tmpl")
			//r.SetHTMLTemplate(tmpl)
			c.Redirect(http.StatusFound, "/users/"+strconv.Itoa(cUser.ID))
		}
	})

	// GET /initialize
	r.GET("/initialize", func(c *gin.Context) {
		db.Exec("DELETE FROM users WHERE id > 5000")
		db.Exec("DELETE FROM products WHERE id > 10000")
		db.Exec("DELETE FROM comments WHERE id > 200000")
		db.Exec("DELETE FROM histories WHERE id > 500000")

		db.Exec("DELETE FROM paged_products")
		_, err := db.Exec("INSERT INTO paged_products (id, page, `name`, description, image_path, price, created_at) " +
			"SELECT " +
			"products.id, " +
			"(10000 - products.id) DIV 50, " +
			"products.name, " +
			"products.description, " +
			"products.image_path, " +
			"products.price, " +
			"products.created_at " +
			"FROM products")
		if err != nil {
			c.Error(err)
			return
		}

		db.Exec("DELETE FROM paged_comments")
		_, err = db.Exec("INSERT INTO paged_comments (id, page, product_id, user_id, content, created_at) " +
			"SELECT " +
			"comments.id, " +
			"(10000 - comments.product_id) DIV 50, " +
			"comments.product_id, " +
			"comments.user_id, " +
			"comments.content, " +
			"comments.created_at " +
			"FROM comments")
		if err != nil {
			c.Error(err)
			return
		}

		initializeHistoryMap()
		initializeProdutMap()

		c.String(http.StatusOK, "Finish")
	})

	pprof.Register(r)
	r.Run(":8080")
}

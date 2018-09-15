package main

import (
	"database/sql"

	"html"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"unicode/utf8"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/contrib/sessions"

	//"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

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
		// shorten description and comment
		var sProducts []PagedProductWithComments
		for _, p := range products {
			if utf8.RuneCountInString(p.Description) > 70 {
				p.Description = string([]rune(p.Description)[:70]) + "…"
			}

			var newCW []CommentWriter
			for _, c := range p.Comments {
				if utf8.RuneCountInString(c.Content) > 25 {
					c.Content = string([]rune(c.Content)[:25]) + "…"
				}
				newCW = append(newCW, c)
			}
			p.Comments = newCW
			sProducts = append(sProducts, p)
		}

		//r.SetHTMLTemplate(template.Must(template.ParseFiles(layout, "templates/index.tmpl")))
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"CurrentUser": cUser,
			"Products":    sProducts,
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

		var contentsBuffer []byte
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
						<a href="/products/` + pID + `">` + html.EscapeString(p.Name) + `</a>
					</div>
					<div class="panel-body">
						<a href="/products/` + pID + `"><img src="` + p.ImagePath + `" class="img-responsive" /></a>
						<h4>価格</h4>
						<p>` + strconv.Itoa(p.Price) + `円</p>
						<h4>商品説明</h4>
						<p>` + html.EscapeString(p.Description) + `</p>
						<h4>購入日時</h4>
						<p>` + p.CreatedAt + `</p>
					</div>
			`)...)
			if user.ID == cUser.ID {
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

		c.String(http.StatusOK, "Finish")
	})

	pprof.Register(r)
	r.Run(":8080")
}

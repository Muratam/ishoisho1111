package main

import (
	"strconv"
	"time"

	"github.com/gin-gonic/contrib/sessions"
)

// User model
type User struct {
	ID        int
	Name      string
	Email     string
	Password  string
	LastLogin string
}

type History struct {
	Product
	UserID    int
	CreatedAt string
}

func authenticate(email string, password string) (User, bool) {
	var u User
	err := db.QueryRow("SELECT * FROM users WHERE email = ? LIMIT 1", email).Scan(&u.ID, &u.Name, &u.Email, &u.Password, &u.LastLogin)
	if err != nil {
		return u, false
	}
	result := password == u.Password
	return u, result
}

func notAuthenticated(session sessions.Session) bool {
	if session == nil {
		return true
	}
	uid := session.Get("uid")
	if uid == nil {
		return true
	}
	return !(uid.(int) > 0)
}

func getUser(uid int) User {
	u := User{}
	r := db.QueryRow("SELECT * FROM users WHERE id = ? LIMIT 1", uid)
	err := r.Scan(&u.ID, &u.Name, &u.Email, &u.Password, &u.LastLogin)
	if err != nil {
		return u
	}

	return u
}

func currentUser(session sessions.Session) User {
	uid := session.Get("uid")
	u := User{}
	r := db.QueryRow("SELECT * FROM users WHERE id = ? LIMIT 1", uid)
	err := r.Scan(&u.ID, &u.Name, &u.Email, &u.Password, &u.LastLogin)
	if err != nil {
		return u
	}

	return u
}
func unsafeParseDate(date string) (time.Time, error) {
	year := (((int(date[0])-'0')*10+int(date[1])-'0')*10+int(date[2])-'0')*10 + int(date[3]) - '0'
	month := time.Month((int(date[5])-'0')*10 + int(date[6]) - '0')
	day := (int(date[8])-'0')*10 + int(date[9]) - '0'
	hour := (int(date[11])-'0')*10 + int(date[12]) - '0'
	minute := (int(date[14])-'0')*10 + int(date[15]) - '0'
	second := (int(date[17])-'0')*10 + int(date[18]) - '0'
	return time.Date(year, month, day, hour, minute, second, 0, time.UTC), nil
}

// BuyingHistory : products which user had bought
func (u *User) BuyingHistory() (products []Product) {
	ps_, ok := historyMap.Load(u.ID)
	if !ok {
		return nil
	}
	ps := ps_.([]Product)
	products = make([]Product, len(ps))
	for i, v := range ps {
		products[i] = v
	}
	return products
}

// BuyProduct : buy product
func (u *User) BuyProduct(pid string) {
	db.Exec(
		"INSERT INTO histories (product_id, user_id, created_at) VALUES (?, ?, ?)",
		pid, u.ID, time.Now())
	ps_, ok := historyMap.Load(u.ID)
	var ps []Product
	if !ok {
		ps = []Product{}
	} else {
		ps = ps_.([]Product)
	}
	ipid, _ := strconv.Atoi(pid)
	p_, _ := productMap.Load(ipid)
	p := p_.(Product)
	fmt := "2006-01-02 15:04:05"
	p.CreatedAt = (time.Now().Add(9 * time.Hour)).Format(fmt)
	historyMap.Store(u.ID, append([]Product{p}, ps...))
}

// CreateComment : create comment to the product
func (u *User) CreateComment(pid string, content string) {
	ipid, _ := strconv.Atoi(pid)
	db.Exec(
		"INSERT INTO paged_comments (page, product_id, user_id, content, created_at) VALUES (?, ?, ?, ?, ?)",
		(10000-ipid)/50, pid, u.ID, content, time.Now())
}

func (u *User) UpdateLastLogin() {
	db.Exec("UPDATE users SET last_login = ? WHERE id = ?", time.Now(), u.ID)
}

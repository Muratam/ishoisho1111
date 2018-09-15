package main

import "log"

// Product Model
type Product struct {
	ID          int
	Name        string
	Description string
	ImagePath   string
	Price       int
	CreatedAt   string
}

// PagedProductWithComments Model
type PagedProductWithComments struct {
	ID           int
	Page         int
	Name         string
	Description  string
	ImagePath    string
	Price        int
	CreatedAt    string
	CommentCount int
	Comments     []CommentWriter
}

// CommentWriter Model
type CommentWriter struct {
	Content string
	Writer  string
}

type PagedCommentWriter struct {
	PID int
	CommentWriter
}

func getProduct(pid int) Product {
	p := Product{}
	row := db.QueryRow("SELECT * FROM products WHERE id = ? LIMIT 1", pid)
	err := row.Scan(&p.ID, &p.Name, &p.Description, &p.ImagePath, &p.Price, &p.CreatedAt)
	if err != nil {
		panic(err.Error())
	}

	return p
}

func getProductsWithCommentsAt(page int) (ret []PagedProductWithComments) {
	// select 50 products with offset page*50
	products := pagedProducts[page]

	productMap := make(map[int]PagedProductWithComments)
	for _, p := range products {
		productMap[p.ID] = p
	}

	tmp := pagedComments[page]
	cs := make([]PagedCommentWriter, len(tmp))
	for i, v := range tmp {
		cs[len(tmp) - i - 1] = v
	}
	for _, c := range cs {
		p := productMap[c.PID]
		p.CommentCount += 1
		if p.CommentCount <= 5 {
			p.Comments = append(p.Comments, c.CommentWriter)
		}
		productMap[p.ID] = p
	}

	for i, p := range products {
		products[i] = productMap[p.ID]
	}

	return products
}

func (p *Product) isBought(uid int) bool {
	var count int
	log.Print(uid)
	log.Print(p.ID)
	err := db.QueryRow(
		"SELECT count(*) as count FROM histories WHERE product_id = ? AND user_id = ?",
		p.ID, uid,
	).Scan(&count)
	if err != nil {
		panic(err.Error())
	}

	return count > 0
}

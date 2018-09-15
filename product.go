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
	var err error
	tx, err := db.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
			if err != nil {
				ret = nil
			}
		}
	}()

	// select 50 products with offset page*50
	products := []PagedProductWithComments{}
	prows, err := tx.Query("SELECT * FROM paged_products WHERE page = ? ORDER BY id DESC", page)
	if err != nil {
		return nil
	}

	defer prows.Close()
	for prows.Next() {
		p := PagedProductWithComments{}
		err = prows.Scan(&p.ID, &p.Page, &p.Name, &p.Description, &p.ImagePath, &p.Price, &p.CreatedAt)
		if err != nil {
			return nil
		}
		p.Comments = []CommentWriter{}
		products = append(products, p)
	}

	productMap := make(map[int]PagedProductWithComments)
	for _, p := range products {
		productMap[p.ID] = p
	}

	crows, err := tx.Query("SELECT product_id,content,name FROM paged_comments AS c INNER JOIN users AS u ON c.user_id = u.id "+
		"WHERE c.page = ? ORDER BY c.created_at DESC", page)
	if err != nil {
		return nil
	}

	defer crows.Close()
	for crows.Next() {
		var pid int
		cw := CommentWriter{}
		err = crows.Scan(&pid, &cw.Content, &cw.Writer)
		if err != nil {
			return nil
		}

		p := productMap[pid]
		p.CommentCount += 1
		if p.CommentCount <= 5 {
			p.Comments = append(p.Comments, cw)
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

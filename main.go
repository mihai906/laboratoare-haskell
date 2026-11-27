// Laboratorul 7 - cele 4 exerciții de la final.
// Tot codul este în acest fișier; go.mod și go.sum descriu dependențele.
//
// Rulare SQL:       go run main.go
// Rulare GORM:      go run main.go -store gorm -debug
// Demo exercițiu 4: go run main.go -demo-zero
//
// API: GET/POST /products și GET/PUT/DELETE /products/:id.
// GET /products?max_price=1000&page=1&limit=10
// POST/PUT: {"name":"Monitor","price":900,"stock":0}
// Reguli: nume nevid, price > 0, stock >= 0; page=1, limit=10 implicit.
// Răspuns listă: {"products":[...],"total":N,"page":1,"limit":10}.
// Datele persistă în products.db. Repornește cu aceeași cale -db pentru verificare.

package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/glebarez/go-sqlite"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Modelul și tipurile comune celor patru exerciții.
var ErrNotFound = errors.New("produs inexistent")

// Product este modelul comun celor două implementări de stocare.
type Product struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price"`
	Stock int    `json:"stock"`
}

type ListOptions struct {
	MaxPrice *int
	Page     int
	Limit    int
}

func (o ListOptions) Validate() error {
	if o.Page < 1 || o.Limit < 1 {
		return errors.New("page și limit trebuie să fie numere întregi pozitive")
	}
	if o.MaxPrice != nil && *o.MaxPrice < 0 {
		return errors.New("max_price trebuie să fie un număr întreg nenegativ")
	}
	if o.Page-1 > math.MaxInt/o.Limit {
		return errors.New("paginarea depășește intervalul numeric acceptat")
	}
	return nil
}

func (o ListOptions) Offset() int { return (o.Page - 1) * o.Limit }

type ProductPage struct {
	Products []Product `json:"products"`
	Total    int64     `json:"total"`
	Page     int       `json:"page"`
	Limit    int       `json:"limit"`
}

// Handler-ele folosesc aceeași interfață indiferent de stocarea aleasă.
type ProductStore interface {
	List(ListOptions) (ProductPage, error)
	Get(int64) (Product, error)
	Create(string, int, int) (Product, error)
	Update(int64, string, int, int) (Product, error)
	Delete(int64) error
	Close() error
}

func initialProducts() []Product {
	return []Product{
		{Name: "Laptop", Price: 3500, Stock: 5},
		{Name: "Mouse", Price: 100, Stock: 20},
		{Name: "Tastatură", Price: 250, Stock: 10},
	}
}

func storageError(operation string, err error) error {
	return fmt.Errorf("%s: %w", operation, err)
}

// Exercițiul 1: schema, inițializarea și stocarea database/sql.
const schema = `
CREATE TABLE IF NOT EXISTS products (
    id    INTEGER PRIMARY KEY AUTOINCREMENT,
    name  TEXT    NOT NULL,
    price INTEGER NOT NULL CHECK (price > 0),
    stock INTEGER NOT NULL CHECK (stock >= 0)
);`

// initializeDatabase este folosită de ambele stocări. Nu folosim AutoMigrate.
func initializeDatabase(db *sql.DB) error {
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		return storageError("conectare", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // După Commit, Rollback nu mai modifică nimic.
	if _, err := tx.Exec(schema); err != nil {
		return storageError("creare tabelă", err)
	}
	var count int64
	if err := tx.QueryRow("SELECT COUNT(*) FROM products").Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		for _, p := range initialProducts() {
			if _, err := tx.Exec("INSERT INTO products (name, price, stock) VALUES (?, ?, ?)",
				p.Name, p.Price, p.Stock); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

type SQLProductStore struct{ db *sql.DB }

func NewSQLProductStore(path string) (*SQLProductStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := initializeDatabase(db); err != nil {
		db.Close()
		return nil, err
	}
	return &SQLProductStore{db: db}, nil
}

func (s *SQLProductStore) Close() error { return s.db.Close() }

// Exercițiul 3: filtrare și paginare în SQL.
// COUNT folosește același filtru; totalul este calculat înainte de LIMIT/OFFSET.
func (s *SQLProductStore) List(o ListOptions) (ProductPage, error) {
	if err := o.Validate(); err != nil {
		return ProductPage{}, err
	}
	result := ProductPage{Products: []Product{}, Page: o.Page, Limit: o.Limit}
	where := ""
	args := []any{}
	if o.MaxPrice != nil {
		where = " WHERE price <= ?"
		args = append(args, *o.MaxPrice)
	}
	// COUNT și SELECT citesc aceeași stare chiar dacă apar scrieri concurente.
	tx, err := s.db.Begin()
	if err != nil {
		return ProductPage{}, err
	}
	defer tx.Rollback()
	if err := tx.QueryRow("SELECT COUNT(*) FROM products"+where, args...).Scan(&result.Total); err != nil {
		return ProductPage{}, err
	}
	query := "SELECT id, name, price, stock FROM products" + where + " ORDER BY id LIMIT ? OFFSET ?"
	args = append(args, o.Limit, o.Offset())
	rows, err := tx.Query(query, args...)
	if err != nil {
		return ProductPage{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Stock); err != nil {
			return ProductPage{}, err
		}
		result.Products = append(result.Products, p)
	}
	if err := rows.Err(); err != nil {
		return ProductPage{}, err
	}
	if err := rows.Close(); err != nil {
		return ProductPage{}, err
	}
	if err := tx.Commit(); err != nil {
		return ProductPage{}, err
	}
	return result, nil
}

func (s *SQLProductStore) Get(id int64) (Product, error) {
	var p Product
	err := s.db.QueryRow("SELECT id, name, price, stock FROM products WHERE id = ?", id).
		Scan(&p.ID, &p.Name, &p.Price, &p.Stock)
	if errors.Is(err, sql.ErrNoRows) {
		return Product{}, ErrNotFound
	}
	return p, err
}

func (s *SQLProductStore) Create(name string, price, stock int) (Product, error) {
	res, err := s.db.Exec("INSERT INTO products (name, price, stock) VALUES (?, ?, ?)", name, price, stock)
	if err != nil {
		return Product{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Product{}, err
	}
	return Product{ID: id, Name: name, Price: price, Stock: stock}, nil
}

func (s *SQLProductStore) Update(id int64, name string, price, stock int) (Product, error) {
	res, err := s.db.Exec("UPDATE products SET name = ?, price = ?, stock = ? WHERE id = ?", name, price, stock, id)
	if err != nil {
		return Product{}, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return Product{}, err
	}
	if n == 0 {
		return Product{}, ErrNotFound
	}
	return Product{ID: id, Name: name, Price: price, Stock: stock}, nil
}

func (s *SQLProductStore) Delete(id int64) error {
	res, err := s.db.Exec("DELETE FROM products WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Exercițiul 2: endpoint-uri CRUD, validare și tratarea erorilor.
type API struct{ store ProductStore }

func NewRouter(store ProductStore) *gin.Engine {
	a := &API{store: store}
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.GET("/products", a.listProducts)
	r.GET("/products/:id", a.getProduct)
	r.POST("/products", a.createProduct)
	r.PUT("/products/:id", a.updateProduct)
	r.DELETE("/products/:id", a.deleteProduct)
	return r
}

func badRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": message})
}

func internalError(c *gin.Context, err error) {
	log.Printf("eroare internă: %v", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "eroare internă"})
}

func storageFailure(c *gin.Context, err error) {
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": ErrNotFound.Error()})
		return
	}
	internalError(c, err)
}

func parseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		badRequest(c, "id trebuie să fie un număr întreg pozitiv")
		return 0, false
	}
	return id, true
}

func parseListOptions(c *gin.Context) (ListOptions, error) {
	o := ListOptions{Page: 1, Limit: 10}
	for _, field := range []struct {
		name   string
		target *int
	}{{"page", &o.Page}, {"limit", &o.Limit}} {
		if raw, present := c.GetQueryArray(field.name); present {
			if len(raw) != 1 {
				return o, fmt.Errorf("%s trebuie transmis o singură dată", field.name)
			}
			n, err := strconv.Atoi(raw[0])
			if err != nil {
				return o, fmt.Errorf("%s trebuie să fie număr întreg", field.name)
			}
			*field.target = n
		}
	}
	if raw, present := c.GetQueryArray("max_price"); present {
		if len(raw) != 1 {
			return o, errors.New("max_price trebuie transmis o singură dată")
		}
		n, err := strconv.Atoi(raw[0])
		if err != nil {
			return o, errors.New("max_price trebuie să fie număr întreg")
		}
		o.MaxPrice = &n
	}
	return o, o.Validate()
}

type productInput struct {
	Name  string `json:"name" binding:"required"`
	Price *int   `json:"price" binding:"required"`
	Stock *int   `json:"stock" binding:"required"`
}

func parseProduct(c *gin.Context) (productInput, bool) {
	var input productInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, "JSON invalid; sunt obligatorii name, price și stock")
		return input, false
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || *input.Price <= 0 || *input.Stock < 0 {
		badRequest(c, "name trebuie să fie nevid, price > 0 și stock >= 0")
		return input, false
	}
	return input, true
}

func (a *API) listProducts(c *gin.Context) {
	options, err := parseListOptions(c)
	if err != nil {
		badRequest(c, err.Error())
		return
	}
	page, err := a.store.List(options)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, page)
}

func (a *API) getProduct(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	p, err := a.store.Get(id)
	if err != nil {
		storageFailure(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (a *API) createProduct(c *gin.Context) {
	input, ok := parseProduct(c)
	if !ok {
		return
	}
	p, err := a.store.Create(input.Name, *input.Price, *input.Stock)
	if err != nil {
		internalError(c, err)
		return
	}
	c.Header("Location", fmt.Sprintf("/products/%d", p.ID))
	c.JSON(http.StatusCreated, p)
}

func (a *API) updateProduct(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	input, ok := parseProduct(c)
	if !ok {
		return
	}
	p, err := a.store.Update(id, input.Name, *input.Price, *input.Stock)
	if err != nil {
		storageFailure(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (a *API) deleteProduct(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := a.store.Delete(id); err != nil {
		storageFailure(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Exercițiul 4: a doua stocare, folosind GORM.
type GormProductStore struct{ db *gorm.DB }

func NewGormProductStore(path string) (*GormProductStore, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if err := initializeDatabase(sqlDB); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return &GormProductStore{db: db}, nil
}

// Se apelează la pornire, înainte ca handler-ele să primească cereri.
func (s *GormProductStore) EnableDebug() { s.db = s.db.Debug() }

func (s *GormProductStore) Close() error {
	db, err := s.db.DB()
	if err != nil {
		return err
	}
	return db.Close()
}

func (s *GormProductStore) List(o ListOptions) (ProductPage, error) {
	if err := o.Validate(); err != nil {
		return ProductPage{}, err
	}
	result := ProductPage{Products: []Product{}, Page: o.Page, Limit: o.Limit}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		query := func() *gorm.DB {
			q := tx.Model(&Product{})
			if o.MaxPrice != nil {
				q = q.Where("price <= ?", *o.MaxPrice)
			}
			return q
		}
		// COUNT se execută înainte de LIMIT/OFFSET; filtrul este identic.
		if err := query().Count(&result.Total).Error; err != nil {
			return err
		}
		return query().Order("id").Limit(o.Limit).Offset(o.Offset()).Find(&result.Products).Error
	})
	return result, err
}

func (s *GormProductStore) Get(id int64) (Product, error) {
	var p Product
	err := s.db.First(&p, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Product{}, ErrNotFound
	}
	return p, err
}

func (s *GormProductStore) Create(name string, price, stock int) (Product, error) {
	p := Product{Name: name, Price: price, Stock: stock}
	if err := s.db.Create(&p).Error; err != nil {
		return Product{}, err
	}
	return p, nil
}

func (s *GormProductStore) Update(id int64, name string, price, stock int) (Product, error) {
	// Un map include și valoarea 0; Updates(Product{Stock: 0}) ar omite-o.
	res := s.db.Model(&Product{}).Where("id = ?", id).
		Updates(map[string]any{"name": name, "price": price, "stock": stock})
	if res.Error != nil {
		return Product{}, res.Error
	}
	if res.RowsAffected == 0 {
		return Product{}, ErrNotFound
	}
	return Product{ID: id, Name: name, Price: price, Stock: stock}, nil
}

func (s *GormProductStore) Delete(id int64) error {
	res := s.db.Delete(&Product{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Exercițiul 4: demonstrație pe un fișier temporar, fără a modifica products.db.
// Updates cu structură omite valorile zero. Un map include explicit stock=0.
func demoZeroStock() error {
	file, err := os.CreateTemp("", "laborator7-*.db")
	if err != nil {
		return err
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		os.Remove(path)
		return err
	}
	defer os.Remove(path)

	store, err := NewGormProductStore(path)
	if err != nil {
		return err
	}
	defer store.Close()
	store.EnableDebug()
	p, err := store.Create("Demo", 100, 5)
	if err != nil {
		return err
	}

	// Varianta greșită: coloana stock lipsește din SQL-ul afișat de Debug().
	err = store.db.Model(&Product{}).Where("id = ?", p.ID).
		Updates(Product{Name: "Demo", Price: 100, Stock: 0}).Error
	if err != nil {
		return err
	}
	p, err = store.Get(p.ID)
	if err != nil {
		return err
	}
	if p.Stock != 5 {
		return fmt.Errorf("demo: stoc neașteptat: %d", p.Stock)
	}
	log.Printf("Updates cu structură: stock=%d; valoarea 0 a fost omisă", p.Stock)

	// Varianta corectată: metoda Update folosește Updates cu un map.
	if _, err := store.Update(p.ID, p.Name, p.Price, 0); err != nil {
		return err
	}
	p, err = store.Get(p.ID)
	if err != nil {
		return err
	}
	if p.Stock != 0 {
		return fmt.Errorf("demo: stocul 0 nu a fost salvat")
	}
	log.Printf("Updates cu map: stock=%d; valoarea 0 a fost salvată", p.Stock)
	return nil
}

// Pornirea API-ului cu oricare dintre cele două stocări.
func run() error {
	backend := flag.String("store", "sql", "stocare: sql sau gorm")
	path := flag.String("db", "products.db", "fișierul SQLite")
	addr := flag.String("addr", "127.0.0.1:8080", "adresa HTTP")
	debug := flag.Bool("debug", false, "afișează interogările GORM")
	demo := flag.Bool("demo-zero", false, "demonstrația stock=0 din exercițiul 4")
	flag.Parse()
	if *demo {
		return demoZeroStock()
	}

	var store ProductStore
	var err error
	switch *backend {
	case "sql":
		store, err = NewSQLProductStore(*path)
	case "gorm":
		var s *GormProductStore
		s, err = NewGormProductStore(*path)
		if err == nil {
			if *debug {
				s.EnableDebug()
			}
			store = s
		}
	default:
		return fmt.Errorf("alege -store sql sau -store gorm")
	}
	if err != nil {
		return err
	}
	defer store.Close()
	return NewRouter(store).Run(*addr)
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

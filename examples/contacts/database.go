package main

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

// Contact represents a contact record in the database
type Contact struct {
	ID        int64
	FirstName string
	LastName  string
	Address   string
	City      string
	State     string
	Zip       string
	Phone     string
	Email     string
}

// InitDatabase opens or creates the SQLite database
func InitDatabase(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// CreateContactsTable creates the contacts table if it doesn't exist
func CreateContactsTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS contacts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		first_name TEXT NOT NULL,
		last_name TEXT NOT NULL,
		address TEXT,
		city TEXT,
		state TEXT,
		zip TEXT,
		phone TEXT,
		email TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_first_name ON contacts(first_name);
	CREATE INDEX IF NOT EXISTS idx_last_name ON contacts(last_name);
	CREATE INDEX IF NOT EXISTS idx_city ON contacts(city);
	CREATE INDEX IF NOT EXISTS idx_state ON contacts(state);
	`
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create contacts table: %w", err)
	}
	return nil
}

// SeedDatabase populates the database with initial contacts if empty
func SeedDatabase(db *sql.DB) error {
	// Check if table has any records
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM contacts").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count contacts: %w", err)
	}

	// Only seed if table is empty
	if count > 0 {
		return nil
	}

	// Insert seed contacts
	for _, c := range SeedContacts {
		if err := InsertContact(db, &c); err != nil {
			return fmt.Errorf("failed to seed contact %s %s: %w", c.FirstName, c.LastName, err)
		}
	}

	return nil
}

// GetAllContacts retrieves all contacts with optional sorting
func GetAllContacts(db *sql.DB, sortBy string, sortAsc bool) ([]*Contact, error) {
	// Validate sort column
	validColumns := map[string]string{
		"first_name": "first_name",
		"last_name":  "last_name",
		"city":       "city",
		"state":      "state",
		"phone":      "phone",
	}

	orderColumn := "last_name" // default
	if col, ok := validColumns[sortBy]; ok {
		orderColumn = col
	}

	orderDir := "ASC"
	if !sortAsc {
		orderDir = "DESC"
	}

	query := fmt.Sprintf(`
		SELECT id, first_name, last_name, address, city, state, zip, phone, email
		FROM contacts
		ORDER BY %s %s, last_name ASC, first_name ASC
	`, orderColumn, orderDir)

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query contacts: %w", err)
	}
	defer rows.Close()

	var contacts []*Contact
	for rows.Next() {
		c := &Contact{}
		err := rows.Scan(&c.ID, &c.FirstName, &c.LastName, &c.Address, &c.City, &c.State, &c.Zip, &c.Phone, &c.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to scan contact: %w", err)
		}
		contacts = append(contacts, c)
	}

	return contacts, nil
}

// GetContact retrieves a single contact by ID
func GetContact(db *sql.DB, id int64) (*Contact, error) {
	query := `
		SELECT id, first_name, last_name, address, city, state, zip, phone, email
		FROM contacts
		WHERE id = ?
	`
	c := &Contact{}
	err := db.QueryRow(query, id).Scan(&c.ID, &c.FirstName, &c.LastName, &c.Address, &c.City, &c.State, &c.Zip, &c.Phone, &c.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("contact not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}
	return c, nil
}

// InsertContact adds a new contact to the database
func InsertContact(db *sql.DB, c *Contact) error {
	query := `
		INSERT INTO contacts (first_name, last_name, address, city, state, zip, phone, email)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := db.Exec(query, c.FirstName, c.LastName, c.Address, c.City, c.State, c.Zip, c.Phone, c.Email)
	if err != nil {
		return fmt.Errorf("failed to insert contact: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	c.ID = id

	return nil
}

// UpdateContact updates an existing contact in the database
func UpdateContact(db *sql.DB, c *Contact) error {
	query := `
		UPDATE contacts
		SET first_name = ?, last_name = ?, address = ?, city = ?, state = ?, zip = ?, phone = ?, email = ?
		WHERE id = ?
	`
	result, err := db.Exec(query, c.FirstName, c.LastName, c.Address, c.City, c.State, c.Zip, c.Phone, c.Email, c.ID)
	if err != nil {
		return fmt.Errorf("failed to update contact: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("contact not found: %d", c.ID)
	}

	return nil
}

// DeleteContact removes a contact from the database
func DeleteContact(db *sql.DB, id int64) error {
	query := "DELETE FROM contacts WHERE id = ?"
	result, err := db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete contact: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("contact not found: %d", id)
	}

	return nil
}

// SearchContacts searches contacts by any field
func SearchContacts(db *sql.DB, query string, sortBy string, sortAsc bool) ([]*Contact, error) {
	// Validate sort column
	validColumns := map[string]string{
		"first_name": "first_name",
		"last_name":  "last_name",
		"city":       "city",
		"state":      "state",
		"phone":      "phone",
	}

	orderColumn := "last_name"
	if col, ok := validColumns[sortBy]; ok {
		orderColumn = col
	}

	orderDir := "ASC"
	if !sortAsc {
		orderDir = "DESC"
	}

	searchTerm := "%" + strings.ToLower(query) + "%"

	sqlQuery := fmt.Sprintf(`
		SELECT id, first_name, last_name, address, city, state, zip, phone, email
		FROM contacts
		WHERE LOWER(first_name) LIKE ?
		   OR LOWER(last_name) LIKE ?
		   OR LOWER(address) LIKE ?
		   OR LOWER(city) LIKE ?
		   OR LOWER(state) LIKE ?
		   OR LOWER(zip) LIKE ?
		   OR LOWER(phone) LIKE ?
		   OR LOWER(email) LIKE ?
		ORDER BY %s %s, last_name ASC, first_name ASC
	`, orderColumn, orderDir)

	rows, err := db.Query(sqlQuery, searchTerm, searchTerm, searchTerm, searchTerm, searchTerm, searchTerm, searchTerm, searchTerm)
	if err != nil {
		return nil, fmt.Errorf("failed to search contacts: %w", err)
	}
	defer rows.Close()

	var contacts []*Contact
	for rows.Next() {
		c := &Contact{}
		err := rows.Scan(&c.ID, &c.FirstName, &c.LastName, &c.Address, &c.City, &c.State, &c.Zip, &c.Phone, &c.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to scan contact: %w", err)
		}
		contacts = append(contacts, c)
	}

	return contacts, nil
}

// ContactCount returns the total number of contacts
func ContactCount(db *sql.DB) int {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM contacts").Scan(&count)
	return count
}

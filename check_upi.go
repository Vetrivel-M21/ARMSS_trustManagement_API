package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	dsn := "root:Vetri@123@tcp(127.0.0.1:3306)/trust_management_db?charset=utf8mb4"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("--- BANK ACCOUNTS ---")
	rows, err := db.Query("SELECT id, bank_name, account_name, upi_id, qr_code_path, is_app_donation_account, is_active FROM bank_accounts")
	if err != nil {
		fmt.Println("Error bank_accounts:", err)
	} else {
		for rows.Next() {
			var id int
			var bank, acc, upi, qr string
			var isApp, isActive bool
			_ = rows.Scan(&id, &bank, &acc, &upi, &qr, &isApp, &isActive)
			fmt.Printf("ID=%d | Bank=%s | AccName=%s | UPI='%s' | QR='%s' | AppDonation=%v | Active=%v\n", id, bank, acc, upi, qr, isApp, isActive)
		}
		rows.Close()
	}

	fmt.Println("\n--- TRUST HOME CONFIGS ---")
	rows, err = db.Query("SELECT id, home_key, home_name, upi_id, qr_code_path, is_active FROM trust_home_configs")
	if err != nil {
		fmt.Println("Error trust_home_configs:", err)
	} else {
		for rows.Next() {
			var id int
			var key, name, upi, qr string
			var isActive bool
			_ = rows.Scan(&id, &key, &name, &upi, &qr, &isActive)
			fmt.Printf("ID=%d | Key=%s | Name=%s | UPI='%s' | QR='%s' | Active=%v\n", id, key, name, upi, qr, isActive)
		}
		rows.Close()
	}

	fmt.Println("\n--- BRANCHES ---")
	rows, err = db.Query("SELECT id, branch_code, name, upi_id, qr_code_path, is_active FROM branches")
	if err != nil {
		fmt.Println("Error branches:", err)
	} else {
		for rows.Next() {
			var id int
			var code, name, upi, qr string
			var isActive bool
			_ = rows.Scan(&id, &code, &name, &upi, &qr, &isActive)
			fmt.Printf("ID=%d | Code=%s | Name=%s | UPI='%s' | QR='%s' | Active=%v\n", id, code, name, upi, qr, isActive)
		}
		rows.Close()
	}
}

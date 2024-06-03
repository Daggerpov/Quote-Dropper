// package main

// import (
// 	"database/sql"
// 	"fmt"
// 	"log"

// 	_ "github.com/lib/pq"
// )

// func main() {
// 	//Get the database URL from the environment variable
// 	dbURL := "postgresql://postgres:fc1DBB1F3Gc25EDCFef1cD2fbFBfdEBg@monorail.proxy.rlwy.net:49806/railway"

// 	// Open a connection to the database
// 	db, err := sql.Open("postgres", dbURL)
// 	if err != nil {
// 		log.Println(err)
// 		log.Fatal(err)
// 	}
// 	defer db.Close()

// 	// Execute ALTER TABLE statement to rename the column
// 	alterStmt := `
// 		ALTER TABLE quotes
// 		RENAME COLUMN "Likes" TO likes;
// 	`
// 	_, err = db.Exec(alterStmt)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	fmt.Println("Column name changed successfully!")
// }

// package main

// import (
// 	"database/sql"
// 	"log"

// 	_ "github.com/lib/pq"
// )

// func main() {
// 	// Get the database URL from the environment variable
// 	dbURL := "postgresql://postgres:fc1DBB1F3Gc25EDCFef1cD2fbFBfdEBg@monorail.proxy.rlwy.net:49806/railway"

// 	// Open a connection to the database
// 	db, err := sql.Open("postgres", dbURL)
// 	if err != nil {
// 		log.Println(err)
// 		log.Fatal(err)
// 	}
// 	defer db.Close()

// 	// Add the Likes column to the quotes table if it doesn't exist
// 	_, err = db.Exec("ALTER TABLE quotes ADD COLUMN IF NOT EXISTS \"Likes\" INTEGER DEFAULT 0")
// 	if err != nil {
// 		log.Println("Error adding Likes column:", err)
// 		log.Fatal(err)
// 	}

// 	// Update the Likes column with random values between 3 and 21
// 	_, err = db.Exec("UPDATE quotes SET \"Likes\" = floor(random() * 19 + 3)")
// 	if err != nil {
// 		log.Println("Error updating Likes column:", err)
// 		log.Fatal(err)
// 	}

// 	log.Println("Updated like counts for all quotes")
// }

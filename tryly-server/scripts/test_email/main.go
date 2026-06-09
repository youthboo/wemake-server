// test_email/main.go — ส่ง email ทดสอบโดยไม่ต้องสร้าง Order ใหม่
// Usage:
//   go run ./scripts/test_email -to=your@email.com -template=SLIP_APPROVED
//   go run ./scripts/test_email -to=your@email.com -template=SLIP_ATTACHED
//   go run ./scripts/test_email -to=your@email.com -template=COMMISSION_INVOICE
//   go run ./scripts/test_email -to=your@email.com -template=COMM_SLIP_ATTACHED

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/yourusername/wemake/internal/mailer"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("env file not loaded: %v", err)
	}

	to := flag.String("to", "", "recipient email (required)")
	templateCode := flag.String("template", "SLIP_APPROVED", "template code: SLIP_ATTACHED | SLIP_APPROVED | COMMISSION_INVOICE | COMM_SLIP_ATTACHED")
	flag.Parse()

	if *to == "" {
		fmt.Println("Usage: go run ./scripts/test_email -to=your@email.com -template=SLIP_APPROVED")
		os.Exit(1)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("DB connect failed: %v", err)
	}
	defer db.Close()

	m := mailer.New(db)

	// dummy data ครอบคลุมทุก placeholder
	data := map[string]string{
		"OrderID":          "51",
		"FactoryName":      "โรงงานทดสอบ ABC",
		"Amount":           "12,500.00",
		"Link":             m.WebURL() + "/orders/51",
		"InvoiceID":        "7",
		"PeriodMonth":      "6",
		"PeriodYear":       "2025",
		"TotalOrders":      "15",
		"CommissionAmount": "3,750.00",
		"VatAmount":        "262.50",
		"GrandTotal":       "4,012.50",
	}

	log.Printf("Sending template=%s to=%s ...", *templateCode, *to)
	if err := m.Send(*templateCode, *to, data, "test", 0); err != nil {
		log.Fatalf("Send failed: %v", err)
	}
	log.Println("✅ Email sent successfully!")
}

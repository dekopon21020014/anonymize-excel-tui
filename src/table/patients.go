package table

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// **PatientsTable 構造体**
type PatientsTable struct {
	DB *sql.DB
}

// **データベースのセットアップ**

func SetupDatabase() *PatientsTable {
	dsn := os.Getenv("DSN")
	dsnDir := filepath.Dir(dsn)

	// ディレクトリがなければ作成
	if err := os.MkdirAll(dsnDir, os.ModePerm); err != nil {
		log.Fatalf("ディレクトリの作成に失敗: %v", err)
	}

	db, err := sql.Open("sqlite3", os.Getenv("DSN"))
	if err != nil {
		log.Fatalf("データベースのオープンに失敗: %v", err)
	}

	// テーブル作成
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS patients (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hashed_id TEXT,
		recorded_date TEXT,
		patient_id TEXT,
		name_kana TEXT,
		name TEXT,		
		gender TEXT,
		birthday TEXT,
		age TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("テーブル作成失敗: %v", err)
	}

	return &PatientsTable{DB: db}
}

/*
  - Insert メソッド * *
    第一引数: hashedID string
    第二引数: record []string
    0: file_name
    1: recorded_date
    2: patient_id
    3: name
    4: number
*/
func (p *PatientsTable) Insert(hashedID string, record []string) {
	if len(record) != 7 {
		log.Fatal("CSVのヘッダ長が異なります，長さ5を期待．")
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")

	insertSQL := `
	INSERT INTO patients (
		hashed_id,
		recorded_date,
		patient_id,
		name_kana,
		name,
		gender,
		birthday,
		age,
		created_at,
		updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`
	_, err := p.DB.Exec(
		insertSQL,
		hashedID,
		record[0], record[1], record[2], record[3], record[4], record[5], record[6],
		now, now,
	)
	if err != nil {
		log.Printf("データ挿入失敗: %v", err)
	} else {
		fmt.Println("データを挿入しました")
	}
}

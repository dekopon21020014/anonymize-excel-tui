package table

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// ** テスト用のデータベースセットアップ **
func setupTestDB(t *testing.T) *PatientsTable {
	t.Helper()

	// テスト用のSQLiteデータベースをメモリ上に作成
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("テスト用DBのオープンに失敗: %v", err)
	}

	// テーブル作成
	createTableSQL := `
	CREATE TABLE patients (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hashed_id TEXT,
		file_name TEXT,		
		recorded_date TEXT,
		patient_id TEXT,
		name TEXT,
		number TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		t.Fatalf("テーブル作成失敗: %v", err)
	}

	return &PatientsTable{DB: db}
}

// ** 正常系のテスト **
func TestInsert_Success(t *testing.T) {
	p := setupTestDB(t)

	// 仮のデータを作成
	hashedID := "abc123"
	record := []string{"file1.mwf", "2025-03-20", "P12345", "John Doe", "56789"}

	// データを挿入
	p.Insert(hashedID, record)

	// データが正しく挿入されているか確認
	var count int
	err := p.DB.QueryRow("SELECT COUNT(*) FROM patients WHERE hashed_id = ?", hashedID).Scan(&count)
	if err != nil {
		t.Fatalf("データ取得失敗: %v", err)
	}

	if count != 1 {
		t.Errorf("期待するデータ件数: 1, 実際の件数: %d", count)
	}
}

// // ** 異常系: record の長さが足りない場合 **
// func TestInsert_Fail_InvalidRecordLength(t *testing.T) {
// 	p := setupTestDB(t)

// 	// 長さが足りないレコードを用意
// 	hashedID := "xyz789"
// 	invalidRecord := []string{"file2.mwf", "2025-03-20"} // 5要素必要なのに2要素しかない

// 	// 標準エラー出力をキャプチャ
// 	old := os.Stderr
// 	_, w, _ := os.Pipe()
// 	os.Stderr = w

// 	// データ挿入（エラーが発生することを期待）
// 	p.Insert(hashedID, invalidRecord)

// 	// エラー出力を取得
// 	w.Close()
// 	os.Stderr = old

// 	// データが挿入されていないことを確認
// 	var count int
// 	err := p.DB.QueryRow("SELECT COUNT(*) FROM patients WHERE hashed_id = ?", hashedID).Scan(&count)
// 	if err != nil {
// 		t.Fatalf("データ取得失敗: %v", err)
// 	}

// 	if count != 0 {
// 		t.Errorf("異常系: 不正なデータが挿入されてしまった")
// 	}
// }

package main

import (
	"anonymize-excel-tui/table"
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"

	"github.com/joho/godotenv"
	"github.com/rivo/tview"
)

var (
	wg            sync.WaitGroup
	app           *tview.Application
	layout        *tview.Flex
	password      string
	saveDir       string
	currentDir    string
	selectedCSV   string
	patientsTable *table.PatientsTable

	// 各種UI
	passwordForm *tview.Form
	csvList      *tview.List
	logView      *tview.TextView
)

// Shift_JIS か UTF-8 かを判定する関数
func detectEncoding(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	buf, err := reader.Peek(512) // 最初の512バイトを取得
	if err != nil && err != io.EOF {
		return "", err
	}

	// UTF-8 の判定
	if bytes.Contains(buf, []byte{0xEF, 0xBB, 0xBF}) || isUTF8(buf) {
		return "UTF-8", nil
	}

	// Shift_JIS の判定
	if isShiftJIS(buf) {
		return "Shift_JIS", nil
	}

	return "Unknown", nil
}

// UTF-8 のバイトシーケンスをチェック
func isUTF8(data []byte) bool {
	return bytes.Contains(data, []byte{0xC2}) || bytes.Contains(data, []byte{0xE3})
}

// Shift_JIS のバイトシーケンスをチェック
func isShiftJIS(data []byte) bool {
	decoder := japanese.ShiftJIS.NewDecoder()
	_, _, err := transform.String(decoder, string(data))
	return err == nil
}

func anonymizeCSV(selectedCSV string) {
	log.Println("Processing CSV:", selectedCSV)

	// 保存用フォルダを作成
	log.Println("保存用フォルダを作成:", saveDir)
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		log.Fatalf("保存フォルダの作成に失敗: %v", err)
		return
	}

	// 入力ファイルを開く
	log.Println("Opening file:", selectedCSV)
	inputFile, err := os.Open(selectedCSV)
	if err != nil {
		log.Println("Error opening input file:", err)
		return
	}
	defer inputFile.Close()

	encodingType, err := detectEncoding(selectedCSV)
	if err != nil {
		log.Println("Error detecting encoding:", err)
		return
	}

	// 出力用のCSVファイルを作成
	outputPath := filepath.Join(saveDir, sanitizeFileName(filepath.Base(selectedCSV)))
	outputFile, err := os.Create(outputPath)
	if err != nil {
		log.Println("Error creating output file:", err)
		return
	}
	defer outputFile.Close()

	// CSVリーダーを文字コードに合わせて作成
	var reader *csv.Reader
	if encodingType == "UTF-8" {
		reader = csv.NewReader(inputFile)
	} else {
		reader = csv.NewReader(transform.NewReader(inputFile, japanese.ShiftJIS.NewDecoder()))
	}

	// CSVライターを作成
	writer := csv.NewWriter(outputFile)
	defer writer.Flush()

	// ヘッダーを読み取り
	header, err := reader.Read()
	if err != nil {
		log.Println("Error reading header:", err)
		return
	}

	// ヘッダーを書き出す
	if err := writer.Write(header); err != nil {
		log.Println("Error writing header:", err)
		return
	}

	// 行数カウンター
	rowCount := 1

	// 一行ずつ処理
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println("Error reading CSV row:", err)
			return
		}

		// データ行の処理
		if len(row) >= 10 { // 最低限必要な列数を確認
			// 患者IDをハッシュ化
			row[1] = sha256Hash(row[1], password)

			// 個人情報を空白に
			row[2] = ""                 // 患者名(半角カナ)
			row[3] = ""                 // 患者名
			row[5] = formatDate(row[5]) // 生年月日をYYYY/MMに変換
			row[8] = ""                 // 所属
			row[9] = ""                 // 保険番号
			row[10] = ""                // 受診日
		}

		// 処理した行を書き出す
		if err := writer.Write(row); err != nil {
			log.Println("Error writing processed row:", err)
			return
		}

		rowCount++

		// 進捗ログ（1000行ごと）
		if rowCount%1000 == 0 {
			log.Printf("Processed %d rows...", rowCount)
		}
	}

	log.Printf("Anonymization complete. Total rows processed: %d", rowCount)
	log.Println("Anonymized file saved to:", outputPath)
	showCompletionMenu()
}

// **SHA256 ハッシュ関数** (変更なし)
func sha256Hash(patientID, password string) string {
	hash := sha256.Sum256([]byte(patientID + password))
	return hex.EncodeToString(hash[:])
}

// **日付のフォーマット関数** (変更なし)
func formatDate(dateStr string) string {
	parsedTime, err := time.Parse("01-02-06", dateStr)
	if err != nil {
		log.Println("日付をパースできませんでした: ", dateStr)
		return "" // 変換できない場合は消す
	}
	return parsedTime.Format("2006/01")
}

// **ファイル名に使えない文字を置換** (変更なし)
func sanitizeFileName(name string) string {
	re := regexp.MustCompile(`[<>:"/\\|?*]`)
	return re.ReplaceAllString(name, "_")
}

// ** excelリストを更新 → CSVリストを更新 **
func updateCSVList() {
	go func() { // 非同期で処理
		csvList.Clear()

		entries, err := os.ReadDir(currentDir)
		if err != nil {
			logView.SetText("ディレクトリ読み取り失敗: " + err.Error())
			return
		}
		for _, entry := range entries {
			filePath := filepath.Join(currentDir, entry.Name())

			if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
				csvList.AddItem("[DIR] "+entry.Name(), "", 0, func() {
					currentDir = filePath
					updateCSVList()
				})
			} else if strings.HasSuffix(strings.ToLower(entry.Name()), ".csv") {
				csvList.AddItem(entry.Name(), "", 0, func() {
					selectedCSV = filePath
					anonymizeCSV(selectedCSV)
				})
			}
		}

		// 親ディレクトリ (..) を追加
		parentDir := filepath.Dir(currentDir)
		csvList.AddItem("[DIR] 前のフォルダに戻る", "", 0, func() {
			currentDir = parentDir
			updateCSVList()
		})

		// 画面を更新 & フォーカス移動
		app.SetFocus(csvList)
		app.Draw()
	}()
}

// ** CSVリストを作成 **
func createCSVList() *tview.List {
	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(true).SetTitle("2. 健診データ(.csv)を選択")
	return list
}

// ** メイン関数 **
func main() {
	// .envファイルを読み込む
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	patientsTable = table.SetupDatabase()
	defer patientsTable.DB.Close()

	setupLogger()

	for { // ユーザが"終了"を選択するまでループ
		// UTC+9の固定タイムゾーン
		jst := time.FixedZone("UTC+9", 9*60*60)

		// 現在の時刻を取得して UTC+9 に変換
		now := time.Now().In(jst)
		saveDir = filepath.Join(os.Getenv("ANNONYMIZED_DATA_DIR"), now.Format("2006-01-02-150405"))

		initializeTUI()
		wg = sync.WaitGroup{}
		if err := app.Run(); err != nil {
			log.Fatalf("failed to start app: %v", err)
		}
		wg.Wait()
	}
}

// ** TUIの初期化 **
func initializeTUI() {
	app = tview.NewApplication()
	currentDir = os.Getenv("CURRENT_DIR")

	// 各画面を作成
	passwordForm = createPasswordForm()
	csvList = createCSVList()
	logView = createLogView()

	// 最初はパスワード画面を表示
	layout = tview.NewFlex().
		AddItem(passwordForm, 0, 1, true).
		AddItem(csvList, 0, 1, false)

	app.SetRoot(layout, true)
}

// ** パスワード入力フォーム ** (変更なし)
func createPasswordForm() *tview.Form {
	form := tview.NewForm()
	if password == "" { // 初回のみパスワード入力を要求
		form.AddPasswordField("パスワード:", "", 20, '*', func(text string) {
			password = text
		})
	} else {
		form.AddTextView("パスワード:", "(前回のパスワードを使用)", 40, 1, false, false)
	}

	form.AddButton("次へ", func() {
		updateCSVList()
	}).
		AddButton("終了", func() {
			app.Stop()
			os.Exit(0)
		})

	form.SetBorder(true).SetTitle("1. パスワード入力")
	return form
}

// ** ログ画面 ** (変更なし)
func createLogView() *tview.TextView {
	logView := tview.NewTextView().SetDynamicColors(true)
	logView.SetBorder(true).SetTitle("ログ")
	return logView
}

// ** 処理完了メニュー ** (変更なし)
func showCompletionMenu() {
	log.Println("complete: ", selectedCSV)
	app.Stop()
	app = tview.NewApplication()

	modal := tview.NewModal().
		SetText(fmt.Sprintf("匿名化したファイルは %s に保存されました。\n続けますか？", saveDir)).
		AddButtons([]string{"続ける", "終了"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "終了" {
				app.Stop()
				os.Exit(0)
			} else {
				app.Stop()
				return
			}
		})

	if err := app.SetRoot(modal, true).Run(); err != nil {
		log.Fatalf("アプリケーションエラー: %v", err)
	}
}

// ** ログ設定 ** (変更なし)
func setupLogger() {
	// 保存フォルダが存在しない場合は作成
	logFileDir := os.Getenv("LOG_FILE_DIR")
	if _, err := os.Stat(logFileDir); os.IsNotExist(err) {
		err := os.MkdirAll(logFileDir, 0755) // フォルダ作成
		if err != nil {
			log.Fatalf("保存フォルダの作成に失敗: %v", err)
		}
	}

	logFile, err := os.OpenFile(
		filepath.Join(logFileDir, os.Getenv("LOG_FILE_NAME")),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0666,
	)
	if err != nil {
		log.Fatalf("ログファイル作成失敗: %v", err)
	}
	log.SetOutput(logFile)
}

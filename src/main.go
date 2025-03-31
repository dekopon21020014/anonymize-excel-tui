package main

import (
	"anonymize-excel-tui/table"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/rivo/tview"
	"github.com/xuri/excelize/v2"
)

var (
	wg            sync.WaitGroup
	app           *tview.Application
	layout        *tview.Flex
	password      string
	saveDir       string
	currentDir    string
	selectedExel  string
	patientsTable *table.PatientsTable

	// 各種UI
	passwordForm *tview.Form
	excelList    *tview.List
	logView      *tview.TextView
)

type ExcelFile struct {
	*excelize.File
}

func (xf *ExcelFile) anonymizeStreaming(outputDir string) error {
	// すべてのシートを取得
	sheets := xf.GetSheetList()
	if len(sheets) == 0 {
		return fmt.Errorf("no sheets found in the Excel file")
	}

	for _, sheetName := range sheets {
		// 出力用のExcelファイルを新規作成
		newFile := excelize.NewFile()
		newSheetIndex, err := newFile.NewSheet(sheetName) // シートを作成anonymizeStreaming
		if err != nil {                                   // シート作成失敗
			log.Println("Error creating new sheet:", err)
			continue
		}
		newFile.SetActiveSheet(newSheetIndex) // 作成したシートをアクティブにする

		// 行を取得（ストリーミング）
		rows, err := xf.Rows(sheetName)
		if err != nil {
			return err
		}
		defer rows.Close()

		// バッファサイズ設定
		bufferSize := 1000
		rowCount := 0
		var processedRows [][]string

		for rows.Next() {
			row, err := rows.Columns()
			if err != nil {
				return err
			}

			if len(row) < 2 {
				processedRows = append(processedRows, row)
				rowCount++
				continue
			}

			patientID := row[1]
			hashedID := sha256Hash(patientID, password)
			// 行を加工
			var additionalCols []string
			for _, col := range row[6:] {
				additionalCols = append(additionalCols, fmt.Sprintf("%v", col)) // 文字列に変換
			}
			processedRow := append([]string{row[0], hashedID, "", "", row[4], formatDate(row[5])}, additionalCols...)
			processedRows = append(processedRows, processedRow)
			rowCount++

			// バッファがいっぱいになったら書き戻す
			if rowCount%bufferSize == 0 {
				if err := writeBuffer(newFile, sheetName, processedRows); err != nil {
					return err
				}
				processedRows = nil // メモリ開放
			}
		}

		// 残りの行を書き戻す
		if len(processedRows) > 0 {
			if err := writeBuffer(newFile, sheetName, processedRows); err != nil {
				return err
			}
		}

		// シート名を元にファイルを保存
		outputPath := filepath.Join(outputDir, fmt.Sprintf("%s.xlsx", sanitizeFileName(sheetName)))
		// **出力先ディレクトリを作成**
		if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
		// **ファイルを保存**
		if err := newFile.SaveAs(outputPath); err != nil {
			return err
		}

		log.Printf("Processed sheet %s: %d rows -> Saved to %s", sheetName, rowCount, outputPath)
	}
	return nil
}

// **バッファのデータを書き戻す関数**
func writeBuffer(xf *excelize.File, sheetName string, processedRows [][]string) error {
	for i, processedRow := range processedRows {
		for j, val := range processedRow {
			colLetter := string(rune('A' + (j % 26))) // A~Z でループするように
			if j >= 26 {
				colLetter = fmt.Sprintf("%c%c", 'A'+(j/26)-1, 'A'+(j%26)) // 2桁カラム対応
			}
			cellRef := fmt.Sprintf("%s%d", colLetter, i+1)
			if err := xf.SetCellValue(sheetName, cellRef, val); err != nil {
				log.Printf("犯人は毛利小五郎")
				return err
			}
		}
	}
	return nil
}

// **ファイル名に使えない文字を置換**
func sanitizeFileName(name string) string {
	re := regexp.MustCompile(`[<>:"/\\|?*]`)
	return re.ReplaceAllString(name, "_")
}

func formatDate(dateStr string) string {
	parsedTime, err := time.Parse("01-02-06", dateStr)
	if err != nil {
		log.Println("日付をパースできませんでした: ", dateStr)
		return "" // 変換できない場合は消す
	}
	return parsedTime.Format("2006/01")
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
		jst := time.FixedZone("UTC+9", 9*60*60)
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

// ** ログ設定 **
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

// ** TUIの初期化 **
func initializeTUI() {
	app = tview.NewApplication()
	// currentDir, _ = os.Getwd()
	currentDir = os.Getenv("CURRENT_DIR")

	// 各画面を作成
	passwordForm = createPasswordForm()
	excelList = createExelList()
	logView = createLogView()

	// **最初はパスワード画面を表示**
	layout = tview.NewFlex().
		AddItem(passwordForm, 0, 1, true).
		AddItem(excelList, 0, 1, false)

	app.SetRoot(layout, true)
}

// ** パスワード入力フォーム **
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
		updateExcelList()
	}).
		AddButton("終了", func() {
			app.Stop()
			os.Exit(0)
		})

	form.SetBorder(true).SetTitle("1. パスワード入力")
	return form
}

func createExelList() *tview.List {
	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(true).SetTitle("2. 健診データ(.xlsx)を選択")
	return list
}

// ** ログ画面 **
func createLogView() *tview.TextView {
	logView := tview.NewTextView().SetDynamicColors(true)
	logView.SetBorder(true).SetTitle("ログ")
	return logView
}

// ** exel選択リストを更新 **
func updateExcelList() {
	go func() { // 非同期で処理
		excelList.Clear()

		entries, err := os.ReadDir(currentDir)
		if err != nil {
			logView.SetText("ディレクトリ読み取り失敗: " + err.Error())
			return
		}
		for _, entry := range entries {
			filePath := filepath.Join(currentDir, entry.Name())

			if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
				excelList.AddItem("[DIR] "+entry.Name(), "", 0, func() {
					currentDir = filePath
					updateExcelList()
				})
			} else if strings.HasSuffix(strings.ToLower(entry.Name()), ".xlsx") {
				excelList.AddItem(entry.Name(), "", 0, func() {
					selectedExel = filePath
					anonymizeExcel(selectedExel)
				})
			}
		}

		// 親ディレクトリ (..) を追加
		parentDir := filepath.Dir(currentDir)
		excelList.AddItem("[DIR] 前のフォルダに戻る", "", 0, func() {
			currentDir = parentDir
			updateExcelList()
		})

		// **[追加] 画面を更新 & フォーカス移動**
		app.SetFocus(excelList)
		app.Draw()
	}()
}

func anonymizeExcel(selectedExcel string) {
	log.Println("ここで実行されてます:", selectedExcel)
	// Excelファイルを開く
	xlFile, err := excelize.OpenFile(selectedExcel)
	if err != nil {
		log.Println("Error opening file:", err)
		return
	}

	excel := &ExcelFile{xlFile}
	err = excel.anonymizeStreaming(saveDir)
	if err != nil {
		log.Println("Error Stringming:", err)
		return
	}
	log.Println("=========================:", saveDir)

	// 保存フォルダが存在しない場合は作成
	if _, err := os.Stat(saveDir); os.IsNotExist(err) {
		err := os.MkdirAll(saveDir, 0755) // フォルダ作成
		if err != nil {
			log.Fatalf("保存フォルダの作成に失敗: %v", err)
		}
	}

	// // 保存
	// outputPath := filepath.Join(saveDir, "anonymizedData.xlsx")
	// if err := excel.SaveAs(outputPath); err != nil {
	// 	log.Println("Error saving file:", err)
	// 	return
	// }

	log.Println("Anonymized file saved to:", saveDir)
	showCompletionMenu()
}

// ** SHA256 ハッシュ関数 **
func sha256Hash(patientID, password string) string {
	hash := sha256.Sum256([]byte(patientID + password))
	return hex.EncodeToString(hash[:])
}

// ** 処理完了メニュー **
func showCompletionMenu() {
	log.Println("complete: ", selectedExel)
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

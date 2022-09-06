package main

import (
	"bufio"
	"context"
	"html/template"
	"net/http"
	"os"
	"regexp"
	"strings"

	"cloud.google.com/go/bigquery"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// global variables
var projectID string
var lootTableName string
var playerTableName string
var bestgearTableName string

func inputForm(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	title := os.Getenv("TITLE")
	if title == "" {
		title = "FFXIV ロット管理"
	}
	tpl, err := template.ParseFiles("input-form.html.tpl")
	if err != nil {
		zap.L().Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := tpl.Execute(w, map[string]interface{}{
		"title": title,
	}); err != nil {
		zap.L().Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func checkDistribution(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// keywords
	lootSuffix := "が戦利品に追加されました。"
	itemPrefix := '\ue0bb'

	var lootItems []string
	lootCandidates := map[string][]string{}

	scanner := bufio.NewScanner(strings.NewReader(r.PostFormValue("log")))
	for scanner.Scan() {
		itemStartRune := strings.IndexRune(scanner.Text(), itemPrefix)
		if strings.Index(scanner.Text(), lootSuffix) != -1 {
			lootItems = append(lootItems, string([]rune(strings.TrimSuffix(scanner.Text(), lootSuffix))[itemStartRune+1:]))
		}
	}

	ctx := context.Background()
	client, err := bigquery.NewClient(ctx, projectID, option.WithScopes(bigquery.Scope, "https://www.googleapis.com/auth/drive"))
	if err != nil {
		zap.L().Fatal(err.Error())
	}

	for _, l := range lootItems {
		zap.L().Info("check candidates for " + l)

		q := client.Query(`
WITH lootCount AS (
	SELECT
		player,
		nickname,
		ifnull(count, priority) AS count,
	FROM ` + "`" + bestgearTableName + "`" + `
	LEFT OUTER JOIN ` + "`" + playerTableName + "`" + ` USING (player)
	LEFT OUTER JOIN (
		SELECT DISTINCT
			player,
			item,
			COUNT(*) OVER (PARTITION BY player, item) AS count,
		FROM ` + "`" + lootTableName + "`" + `
		WHERE item = "` + l + `"
	) USING (player, item)
	WHERE item = "` + l + `"
)
SELECT nickname
FROM lootCount
WHERE count = (SELECT MIN(count) from lootCount)
`)

		it, err := q.Read(ctx)
		if err != nil {
			zap.L().Error(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		for {
			var player struct{ Nickname string }
			err := it.Next(&player)
			if err == iterator.Done {
				break
			}
			if err != nil {
				zap.L().Error(err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			lootCandidates[l] = append(lootCandidates[l], player.Nickname)
		}
	}

	tpl, err := template.ParseFiles("check.html.tpl")
	if err != nil {
		zap.L().Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := tpl.Execute(w, map[string]interface{}{
		"candidates": lootCandidates,
	}); err != nil {
		zap.L().Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func confirmObtaining(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	// keywords
	obtainSuffix := "を手に入れた。"
	itemPrefix := '\ue0bb'

	obtain := []struct {
		Player string
		Item   string
	}{}

	re := regexp.MustCompile(`^(?:\[\d\d?\:\d\d\] )?([^ ]+) `)

	scanner := bufio.NewScanner(strings.NewReader(r.PostFormValue("log")))
	for scanner.Scan() {
		if strings.Index(scanner.Text(), obtainSuffix) != -1 {
			itemStartRune := strings.IndexRune(scanner.Text(), itemPrefix)

			obtain = append(obtain, struct {
				Player string
				Item   string
			}{
				Player: re.FindStringSubmatch(scanner.Text())[1],
				Item:   string([]rune(strings.TrimSuffix(scanner.Text(), obtainSuffix))[itemStartRune-1:]),
			})
		}
	}

	tpl, err := template.ParseFiles("obtain.html.tpl")
	if err != nil {
		zap.L().Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := tpl.Execute(w, map[string]interface{}{
		"obtain": obtain,
	}); err != nil {
		zap.L().Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func submitObtaining(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		zap.L().Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	q := client.Query(strings.ReplaceAll(r.PostFormValue("sql"), "TABLE_NAME", lootTableName))
	job, err := q.Run(ctx)
	if err != nil {
		zap.L().Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	zap.L().Info("submitted obtaining logs. job ID: " + job.ID())

	tpl, err := template.ParseFiles("blank.html.tpl")
	if err != nil {
		zap.L().Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := tpl.Execute(w, map[string]interface{}{}); err != nil {
		zap.L().Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func main() {
	// initiazize logger
	logCfg := zap.NewProductionConfig()
	logCfg.EncoderConfig.TimeKey = "time"
	logCfg.EncoderConfig.LevelKey = "severity"
	logCfg.EncoderConfig.MessageKey = "message"
	logCfg.EncoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	logCfg.EncoderConfig.EncodeLevel = func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		switch l {
		case zapcore.DebugLevel:
			enc.AppendString("DEBUG")
		case zapcore.InfoLevel:
			enc.AppendString("INFO")
		case zapcore.WarnLevel:
			enc.AppendString("WARNING")
		case zapcore.ErrorLevel:
			enc.AppendString("ERROR")
		case zapcore.DPanicLevel:
			enc.AppendString("CRITICAL")
		case zapcore.PanicLevel:
			enc.AppendString("ALERT")
		case zapcore.FatalLevel:
			enc.AppendString("EMERGENCY")
		}
	}

	logger, err := logCfg.Build()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	undo := zap.ReplaceGlobals(logger)
	defer undo()

	// initialize env variables
	projectID = os.Getenv("PROJECT_ID")
	lootTableName = os.Getenv("BQ_TABLE_LOOT")
	playerTableName = os.Getenv("BQ_TABLE_PLAYER")
	bestgearTableName = os.Getenv("BQ_TABLE_BESTGEAR")
	if projectID == "" || lootTableName == "" || playerTableName == "" || bestgearTableName == "" {
		zap.L().Fatal("You MUST specify env PROJECT_ID, BQ_TABLE_LOOT, BQ_TABLE_PLAYER and BQ_TABLE_BESTGEAR.")
	}

	http.HandleFunc("/", inputForm)
	http.HandleFunc("/check", checkDistribution)
	http.HandleFunc("/obtain", confirmObtaining)
	http.HandleFunc("/obtain/submit", submitObtaining)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		zap.L().Info("defaulting to port " + port)
	}

	zap.L().Info("listening on port " + port)
	http.ListenAndServe(":8080", nil)
}

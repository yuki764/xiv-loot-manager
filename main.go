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
	"google.golang.org/api/iterator"
)

func inputForm(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	title := os.Getenv("TITLE")
	if title == "" {
		title = "FFXIV ロット管理"
	}
	tpl, err := template.ParseFiles("input-form.html.tpl")
	if err != nil {
		logger.Fatal(err.Error())
	}
	if err := tpl.Execute(w, map[string]interface{}{
		"title": title,
	}); err != nil {
		logger.Fatal(err.Error())
	}
}

func checkDistribution(w http.ResponseWriter, r *http.Request) {
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

	projectID := os.Getenv("PROJECT_ID")
	lootTableName := os.Getenv("BQ_TABLE_LOOT")
	playerTableName := os.Getenv("BQ_TABLE_PLAYER")
	bestgearTableName := os.Getenv("BQ_TABLE_BESTGEAR")
	if projectID == "" || lootTableName == "" || playerTableName == "" || bestgearTableName == "" {
		logger.Fatal("You MUST specify env PROJECT_ID, BQ_TABLE_LOOT, BQ_TABLE_PLAYER and BQ_TABLE_BESTGEAR.")
	}

	ctx := context.Background()
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		logger.Fatal(err.Error())
	}

	for _, l := range lootItems {
		logger.Info("check candidates for " + l)

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
			logger.Fatal(err.Error())
		}

		for {
			var player struct{ Nickname string }
			err := it.Next(&player)
			if err == iterator.Done {
				break
			}
			if err != nil {
				logger.Fatal(err.Error())
			}
			lootCandidates[l] = append(lootCandidates[l], player.Nickname)
		}
	}

	tpl, err := template.ParseFiles("check.html.tpl")
	if err != nil {
		logger.Fatal(err.Error())
	}
	if err := tpl.Execute(w, map[string]interface{}{
		"candidates": lootCandidates,
	}); err != nil {
		logger.Fatal(err.Error())
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
		logger.Fatal(err.Error())
	}
	if err := tpl.Execute(w, map[string]interface{}{
		"obtain": obtain,
	}); err != nil {
		logger.Fatal(err.Error())
	}
}

func submitObtaining(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	projectID := os.Getenv("PROJECT_ID")
	lootTableName := os.Getenv("BQ_TABLE_LOOT")
	playerTableName := os.Getenv("BQ_TABLE_PLAYER")
	if projectID == "" || lootTableName == "" || playerTableName == "" {
		logger.Fatal("You MUST specify env PROJECT_ID, BQ_TABLE_LOOT and BQ_TABLE_PLAYER.")
	}

	tpl, err := template.ParseFiles("blank.html.tpl")
	if err != nil {
		logger.Fatal(err.Error())
	}
	if err := tpl.Execute(w, map[string]interface{}{}); err != nil {
		logger.Fatal(err.Error())
	}

	ctx := context.Background()
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		logger.Fatal(err.Error())
	}

	q := client.Query(strings.ReplaceAll(r.PostFormValue("sql"), "TABLE_NAME", lootTableName))
	job, err := q.Run(ctx)
	if err != nil {
		logger.Fatal(err.Error())
	}
	logger.Info("submitted obtaining logs. job ID: " + job.ID())
}

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	http.HandleFunc("/", inputForm)
	http.HandleFunc("/check", checkDistribution)
	http.HandleFunc("/obtain", confirmObtaining)
	http.HandleFunc("/obtain/submit", submitObtaining)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		logger.Info("defaulting to port " + port)
	}

	// Start HTTP server.
	logger.Info("listening on port " + port)
	http.ListenAndServe(":8080", nil)
}

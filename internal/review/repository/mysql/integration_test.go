package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"

	"github.com/truongcongminh96/ai-game-review-analyzer/config"
	platformmysql "github.com/truongcongminh96/ai-game-review-analyzer/internal/platform/database/mysql"
	"github.com/truongcongminh96/ai-game-review-analyzer/internal/review/model"
)

func TestMySQLRepositories_Integration(t *testing.T) {
	t.Helper()

	dsn := requireMySQLIntegrationDSN(t)
	testDatabaseDSN, cleanup := createMySQLIntegrationDatabase(t, dsn)
	defer cleanup()

	applyMySQLMigrations(t, testDatabaseDSN)

	db := openMySQLDB(t, testDatabaseDSN)
	defer func() { _ = db.Close() }()

	assertColumnDoesNotExist(t, db, currentDatabaseName(t, db), "analysis_runs", "genre")

	client, err := platformmysql.New(context.Background(), config.Config{
		DatabaseDriver:           config.DatabaseDriverMySQL,
		DatabaseURL:              testDatabaseDSN,
		DatabaseMaxConns:         4,
		DatabaseMinConns:         0,
		DatabaseHealthTimeoutSec: 2,
	})
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	gameRepo := NewGameRepository(client)
	analysisRepo := NewAnalysisRepository(client)

	game, err := gameRepo.UpsertBySteamAppID(context.Background(), model.GameUpsertInput{
		SteamAppID:  "730",
		Title:       "Counter-Strike 2",
		CoverURL:    stringPtr("https://cdn.example/cs2.jpg"),
		Genre:       stringPtr("Action"),
		ReleaseYear: intPtr(2023),
	})
	require.NoError(t, err)
	require.NotNil(t, game)
	require.Equal(t, "Action", game.Genre)

	run, err := analysisRepo.CreateRun(context.Background(), model.CreateAnalysisRunInput{
		GameID:      game.ID,
		ReviewLimit: 25,
		Language:    "english",
	})
	require.NoError(t, err)
	require.NotNil(t, run)

	err = analysisRepo.CompleteRun(context.Background(), model.CompleteAnalysisRunInput{
		RunID:       run.ID,
		ReviewCount: 2,
		ModelName:   "qwen3-coder:30b",
		Insight: &model.Insight{
			Summary:         "Players like the combat loop.",
			PraisedFeatures: []string{"combat", "gunplay"},
			CommonIssues:    []string{"matchmaking"},
			Topics:          []string{"multiplayer"},
			Sentiment: model.SentimentBreakdown{
				Positive: 1,
				Negative: 1,
			},
		},
	})
	require.NoError(t, err)

	assertAnalysisRun(t, db, run.ID)
	assertAnalysisResult(t, db, run.ID)
}

func requireMySQLIntegrationDSN(t *testing.T) string {
	t.Helper()

	if os.Getenv("RUN_MYSQL_INTEGRATION") != "1" {
		t.Skip("set RUN_MYSQL_INTEGRATION=1 to enable MySQL integration tests")
	}

	_ = godotenv.Load(filepath.Join(projectRoot(t), ".env"))

	if dsn := strings.TrimSpace(os.Getenv("MYSQL_INTEGRATION_DSN")); dsn != "" {
		return normalizeMySQLDSN(dsn)
	}

	if strings.EqualFold(strings.TrimSpace(os.Getenv("DATABASE_DRIVER")), config.DatabaseDriverMySQL) {
		if dsn := strings.TrimSpace(os.Getenv("DATABASE_URL")); dsn != "" {
			return normalizeMySQLDSN(dsn)
		}
	}

	t.Fatal("MYSQL_INTEGRATION_DSN or DATABASE_URL with DATABASE_DRIVER=mysql is required")
	return ""
}

func createMySQLIntegrationDatabase(t *testing.T, baseDSN string) (string, func()) {
	t.Helper()

	parsed, err := mysqldriver.ParseDSN(trimMySQLScheme(baseDSN))
	require.NoError(t, err)

	serverConfig := *parsed
	serverConfig.DBName = ""

	serverDB := openMySQLDB(t, serverConfig.FormatDSN())

	databaseName := fmt.Sprintf("ai_game_review_analyzer_it_%d", time.Now().UnixNano())
	_, err = serverDB.Exec("create database `" + databaseName + "` character set utf8mb4 collate utf8mb4_unicode_ci")
	require.NoError(t, err)

	testConfig := *parsed
	testConfig.DBName = databaseName
	testDSN := testConfig.FormatDSN()

	cleanup := func() {
		testDB, err := sql.Open(config.DatabaseDriverMySQL, testDSN)
		if err == nil {
			_ = testDB.Close()
		}

		_, _ = serverDB.Exec("drop database if exists `" + databaseName + "`")
		_ = serverDB.Close()
	}

	return testDSN, cleanup
}

func applyMySQLMigrations(t *testing.T, dsn string) {
	t.Helper()

	sourceURL := (&url.URL{
		Scheme: "file",
		Path:   filepath.ToSlash(filepath.Join(projectRoot(t), "db", "migrations", "mysql")),
	}).String()

	m, err := migrate.New(sourceURL, "mysql://"+trimMySQLScheme(dsn))
	require.NoError(t, err)
	defer func() {
		sourceErr, databaseErr := m.Close()
		require.NoError(t, sourceErr)
		require.NoError(t, databaseErr)
	}()

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		require.NoError(t, err)
	}
}

func openMySQLDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()

	db, err := sql.Open(config.DatabaseDriverMySQL, trimMySQLScheme(dsn))
	require.NoError(t, err)

	require.NoError(t, db.Ping())

	return db
}

func assertColumnDoesNotExist(t *testing.T, db *sql.DB, schemaName string, tableName string, columnName string) {
	t.Helper()

	var count int
	err := db.QueryRow(
		`
			select count(*)
			from information_schema.columns
			where table_schema = ?
			  and table_name = ?
			  and column_name = ?
		`,
		schemaName,
		tableName,
		columnName,
	).Scan(&count)
	require.NoError(t, err)
	require.Zero(t, count)
}

func currentDatabaseName(t *testing.T, db *sql.DB) string {
	t.Helper()

	var name string
	require.NoError(t, db.QueryRow("select database()").Scan(&name))
	require.NotEmpty(t, name)

	return name
}

func assertAnalysisRun(t *testing.T, db *sql.DB, runID string) {
	t.Helper()

	var (
		gameID      string
		reviewLimit int
		language    string
		reviewCount int
		status      string
	)

	err := db.QueryRow(
		`
			select game_id, review_limit, language, review_count, status
			from analysis_runs
			where id = ?
		`,
		runID,
	).Scan(&gameID, &reviewLimit, &language, &reviewCount, &status)
	require.NoError(t, err)
	require.NotEmpty(t, gameID)
	require.Equal(t, 25, reviewLimit)
	require.Equal(t, "english", language)
	require.Equal(t, 2, reviewCount)
	require.Equal(t, "success", status)
}

func assertAnalysisResult(t *testing.T, db *sql.DB, runID string) {
	t.Helper()

	var (
		summary         string
		praisedFeatures []byte
		commonIssues    []byte
		topics          []byte
		modelName       sql.NullString
	)

	err := db.QueryRow(
		`
			select summary, praised_features, common_issues, topics, model_name
			from analysis_results
			where analysis_run_id = ?
		`,
		runID,
	).Scan(&summary, &praisedFeatures, &commonIssues, &topics, &modelName)
	require.NoError(t, err)
	require.Equal(t, "Players like the combat loop.", summary)
	require.True(t, modelName.Valid)
	require.Equal(t, "qwen3-coder:30b", modelName.String)

	require.Equal(t, []string{"combat", "gunplay"}, decodeJSONStringArray(t, praisedFeatures))
	require.Equal(t, []string{"matchmaking"}, decodeJSONStringArray(t, commonIssues))
	require.Equal(t, []string{"multiplayer"}, decodeJSONStringArray(t, topics))
}

func decodeJSONStringArray(t *testing.T, raw []byte) []string {
	t.Helper()

	var items []string
	require.NoError(t, json.Unmarshal(raw, &items))

	return items
}

func projectRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)

	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
}

func normalizeMySQLDSN(dsn string) string {
	trimmed := strings.TrimSpace(dsn)
	if strings.HasPrefix(strings.ToLower(trimmed), "mysql://") {
		return trimmed
	}

	return trimMySQLScheme(trimmed)
}

func trimMySQLScheme(dsn string) string {
	trimmed := strings.TrimSpace(dsn)
	if strings.HasPrefix(strings.ToLower(trimmed), "mysql://") {
		return trimmed[len("mysql://"):]
	}

	return trimmed
}

func stringPtr(value string) *string {
	return &value
}

func intPtr(value int) *int {
	return &value
}

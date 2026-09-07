//go:build integration

package repository

import (
	"context"
	"database/sql"
	"io"
	"os"
	"testing"

	"golang-clean-architecture/internal/dto"
	m "golang-clean-architecture/internal/persistence/model"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	_ "github.com/uptrace/bun/driver/pgdriver"
)

const defaultTestDSN = "postgres://postgres:postgres@127.0.0.1:5432/db?sslmode=disable"

// setupItemRepoTest connects to a real Postgres instance (see readme.md /
// Makefile for `postgres-docker` + `goose-up`) and skips the test if it's
// unreachable, so `go test -tags=integration` stays usable without a DB.
func setupItemRepoTest(t *testing.T) *ItemRepository {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		dsn = defaultTestDSN
	}

	sqldb, err := sql.Open("pg", dsn)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := sqldb.Ping(); err != nil {
		t.Skipf("postgres unreachable at %s (run `make postgres-docker && make goose-up`): %v", dsn, err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })

	db := bun.NewDB(sqldb, pgdialect.New())
	log := logrus.New()
	log.SetOutput(io.Discard)
	return NewItemRepository(db, log)
}

// createTestItem inserts an item with a unique name/SKU (so tests never
// collide with each other or pre-existing rows) and registers cleanup.
func createTestItem(t *testing.T, repo *ItemRepository, marker string, overrides func(*m.Item)) *m.Item {
	t.Helper()

	item := &m.Item{
		Name:     "it-" + marker + "-" + uuid.NewString(),
		Sku:      "SKU-" + uuid.NewString(),
		Currency: "USD",
		Stock:    10,
	}
	if overrides != nil {
		overrides(item)
	}

	id, err := repo.Create(context.Background(), nil, item)
	if err != nil {
		t.Fatalf("failed to create test item: %v", err)
	}
	item.ID = id

	t.Cleanup(func() {
		_ = repo.Delete(context.Background(), nil, item)
	})

	return item
}

func TestItemRepository_Create_Success(t *testing.T) {
	repo := setupItemRepoTest(t)

	item := createTestItem(t, repo, "create", nil)

	assert.Greater(t, item.ID, int64(0))

	found, err := repo.FindById(context.Background(), nil, item.ID)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, item.Name, found.Name)
	assert.Equal(t, item.Sku, found.Sku)
}

func TestItemRepository_FindById_NotFound(t *testing.T) {
	repo := setupItemRepoTest(t)

	found, err := repo.FindById(context.Background(), nil, 0)

	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestItemRepository_Update_Success(t *testing.T) {
	repo := setupItemRepoTest(t)
	item := createTestItem(t, repo, "update", nil)

	item.Name = "updated-" + uuid.NewString()
	item.Stock = 99
	err := repo.Update(context.Background(), nil, item)
	assert.NoError(t, err)

	found, err := repo.FindById(context.Background(), nil, item.ID)
	assert.NoError(t, err)
	assert.Equal(t, item.Name, found.Name)
	assert.Equal(t, int32(99), found.Stock)
}

func TestItemRepository_Update_OmitZeroLeavesOtherFieldsUntouched(t *testing.T) {
	repo := setupItemRepoTest(t)
	item := createTestItem(t, repo, "omitzero", func(i *m.Item) {
		i.Name = "original-" + uuid.NewString()
		i.Currency = "USD"
		i.Stock = 5
	})

	// Only ID + Stock set; Name/Sku/Currency are zero-value and must be
	// omitted from the UPDATE statement rather than clobbering existing data.
	partial := &m.Item{ID: item.ID, Stock: 20}
	err := repo.Update(context.Background(), nil, partial)
	assert.NoError(t, err)

	found, err := repo.FindById(context.Background(), nil, item.ID)
	assert.NoError(t, err)
	assert.Equal(t, item.Name, found.Name)
	assert.Equal(t, item.Sku, found.Sku)
	assert.Equal(t, item.Currency, found.Currency)
	assert.Equal(t, int32(20), found.Stock)
}

func TestItemRepository_Update_NotFound(t *testing.T) {
	repo := setupItemRepoTest(t)

	err := repo.Update(context.Background(), nil, &m.Item{ID: 0, Stock: 1})

	assert.NoError(t, err)
}

func TestItemRepository_Delete_Success(t *testing.T) {
	repo := setupItemRepoTest(t)
	item := createTestItem(t, repo, "delete", nil)

	err := repo.Delete(context.Background(), nil, item)
	assert.NoError(t, err)

	found, err := repo.FindById(context.Background(), nil, item.ID)
	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestItemRepository_Delete_NotFound(t *testing.T) {
	repo := setupItemRepoTest(t)

	err := repo.Delete(context.Background(), nil, &m.Item{ID: 0})

	assert.NoError(t, err)
}

func TestItemRepository_Search_NameFilter(t *testing.T) {
	repo := setupItemRepoTest(t)
	marker := uuid.NewString()

	a := createTestItem(t, repo, "search-name", func(i *m.Item) { i.Name = "Widget-" + marker })
	b := createTestItem(t, repo, "search-name", func(i *m.Item) { i.Name = "Gadget-" + marker })
	_ = createTestItem(t, repo, "search-name-other", nil)

	items, total, err := repo.Search(context.Background(), nil, &dto.SearchItemRequest{
		Name: marker,
		Page: 1,
		Size: 10,
	})

	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	ids := []int64{items[0].ID, items[1].ID}
	assert.ElementsMatch(t, []int64{a.ID, b.ID}, ids)
}

func TestItemRepository_Search_SKUFilter(t *testing.T) {
	repo := setupItemRepoTest(t)
	marker := uuid.NewString()

	a := createTestItem(t, repo, "search-sku", func(i *m.Item) { i.Sku = "SKU-" + marker })

	items, total, err := repo.Search(context.Background(), nil, &dto.SearchItemRequest{
		SKU:  marker,
		Page: 1,
		Size: 10,
	})

	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, a.ID, items[0].ID)
}

func TestItemRepository_Search_Sort(t *testing.T) {
	repo := setupItemRepoTest(t)
	marker := uuid.NewString()

	first := createTestItem(t, repo, "search-sort", func(i *m.Item) { i.Name = "A-" + marker })
	second := createTestItem(t, repo, "search-sort", func(i *m.Item) { i.Name = "B-" + marker })

	itemsAsc, _, err := repo.Search(context.Background(), nil, &dto.SearchItemRequest{
		Name: marker,
		Sort: "name",
		Page: 1,
		Size: 10,
	})
	assert.NoError(t, err)
	assert.Equal(t, []int64{first.ID, second.ID}, []int64{itemsAsc[0].ID, itemsAsc[1].ID})

	itemsDesc, _, err := repo.Search(context.Background(), nil, &dto.SearchItemRequest{
		Name: marker,
		Sort: "-name",
		Page: 1,
		Size: 10,
	})
	assert.NoError(t, err)
	assert.Equal(t, []int64{second.ID, first.ID}, []int64{itemsDesc[0].ID, itemsDesc[1].ID})
}

func TestItemRepository_Search_Pagination(t *testing.T) {
	repo := setupItemRepoTest(t)
	marker := uuid.NewString()

	for i := 0; i < 3; i++ {
		createTestItem(t, repo, "search-page", func(item *m.Item) { item.Name = "Paged-" + marker })
	}

	page1, total, err := repo.Search(context.Background(), nil, &dto.SearchItemRequest{
		Name: marker,
		Page: 1,
		Size: 2,
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, page1, 2)

	page2, total, err := repo.Search(context.Background(), nil, &dto.SearchItemRequest{
		Name: marker,
		Page: 2,
		Size: 2,
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, page2, 1)
}

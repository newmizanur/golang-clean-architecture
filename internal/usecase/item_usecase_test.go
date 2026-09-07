package usecase

import (
	"context"
	"errors"
	"io"
	"testing"

	"golang-clean-architecture/internal/apperror"
	"golang-clean-architecture/internal/dto"
	m "golang-clean-architecture/internal/persistence/model"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

// mockItemRepository is a hand-written mock implementing ItemRepositoryPort.
type mockItemRepository struct {
	findByIdFunc func(ctx context.Context, tx bun.IDB, id int64) (*m.Item, error)
	searchFunc   func(ctx context.Context, tx bun.IDB, search *dto.SearchItemRequest) ([]m.Item, int64, error)
	createFunc   func(ctx context.Context, tx bun.IDB, item *m.Item) (int64, error)
	updateFunc   func(ctx context.Context, tx bun.IDB, item *m.Item) error
	deleteFunc   func(ctx context.Context, tx bun.IDB, item *m.Item) error
}

func (r *mockItemRepository) FindById(ctx context.Context, tx bun.IDB, id int64) (*m.Item, error) {
	return r.findByIdFunc(ctx, tx, id)
}

func (r *mockItemRepository) Search(ctx context.Context, tx bun.IDB, search *dto.SearchItemRequest) ([]m.Item, int64, error) {
	return r.searchFunc(ctx, tx, search)
}

func (r *mockItemRepository) Create(ctx context.Context, tx bun.IDB, item *m.Item) (int64, error) {
	return r.createFunc(ctx, tx, item)
}

func (r *mockItemRepository) Update(ctx context.Context, tx bun.IDB, item *m.Item) error {
	return r.updateFunc(ctx, tx, item)
}

func (r *mockItemRepository) Delete(ctx context.Context, tx bun.IDB, item *m.Item) error {
	return r.deleteFunc(ctx, tx, item)
}

// newTestItemUseCase wires an ItemUseCase against a sqlmock-backed *bun.DB, so
// BeginTx/Commit/Rollback work without a real database connection.
func newTestItemUseCase(t *testing.T) (*ItemUseCase, sqlmock.Sqlmock, *mockItemRepository) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	bunDB := bun.NewDB(sqlDB, pgdialect.New())
	log := logrus.New()
	log.SetOutput(io.Discard)
	repo := &mockItemRepository{}

	uc := NewItemUseCase(bunDB, log, validator.New(), repo)
	return uc, mock, repo
}

// ---- Create tests ----

func TestItemUseCase_Create_Success(t *testing.T) {
	uc, mock, repo := newTestItemUseCase(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	repo.createFunc = func(_ context.Context, _ bun.IDB, item *m.Item) (int64, error) {
		return 42, nil
	}

	req := &dto.CreateItemRequest{Name: "Widget", SKU: "W-1", Currency: "USD", Stock: 5}
	resp, err := uc.Create(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, int64(42), resp.ID)
	assert.Equal(t, "Widget", resp.Name)
	assert.Equal(t, "W-1", resp.SKU)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestItemUseCase_Create_InvalidRequest(t *testing.T) {
	uc, mock, _ := newTestItemUseCase(t)

	resp, err := uc.Create(context.Background(), &dto.CreateItemRequest{})

	assert.Nil(t, resp)
	assert.Equal(t, apperror.ItemErrors.InvalidRequest, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestItemUseCase_Create_TransactionBeginError(t *testing.T) {
	uc, mock, _ := newTestItemUseCase(t)
	mock.ExpectBegin().WillReturnError(errors.New("connection refused"))

	req := &dto.CreateItemRequest{Name: "Widget", SKU: "W-1", Currency: "USD", Stock: 5}
	resp, err := uc.Create(context.Background(), req)

	assert.Nil(t, resp)
	assert.Equal(t, apperror.ItemErrors.FailedToCreateTransaction, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestItemUseCase_Create_RepositoryError(t *testing.T) {
	uc, mock, repo := newTestItemUseCase(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	repo.createFunc = func(_ context.Context, _ bun.IDB, item *m.Item) (int64, error) {
		return 0, errors.New("insert failed")
	}

	req := &dto.CreateItemRequest{Name: "Widget", SKU: "W-1", Currency: "USD", Stock: 5}
	resp, err := uc.Create(context.Background(), req)

	assert.Nil(t, resp)
	assert.Equal(t, apperror.ItemErrors.FailedToCreateItem, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestItemUseCase_Create_CommitError(t *testing.T) {
	uc, mock, repo := newTestItemUseCase(t)
	mock.ExpectBegin()
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	repo.createFunc = func(_ context.Context, _ bun.IDB, item *m.Item) (int64, error) {
		return 1, nil
	}

	req := &dto.CreateItemRequest{Name: "Widget", SKU: "W-1", Currency: "USD", Stock: 5}
	resp, err := uc.Create(context.Background(), req)

	assert.Nil(t, resp)
	assert.Equal(t, apperror.ItemErrors.FailedToCreateItem, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---- Search tests ----

func TestItemUseCase_Search_Success(t *testing.T) {
	uc, _, repo := newTestItemUseCase(t)
	items := []m.Item{{ID: 1, Name: "A"}, {ID: 2, Name: "B"}}

	repo.searchFunc = func(_ context.Context, tx bun.IDB, _ *dto.SearchItemRequest) ([]m.Item, int64, error) {
		assert.Nil(t, tx)
		return items, 2, nil
	}

	resp, total, err := uc.Search(context.Background(), &dto.SearchItemRequest{Page: 1, Size: 10})

	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, resp, 2)
	assert.Equal(t, "A", resp[0].Name)
}

func TestItemUseCase_Search_RepositoryError(t *testing.T) {
	uc, _, repo := newTestItemUseCase(t)

	repo.searchFunc = func(_ context.Context, _ bun.IDB, _ *dto.SearchItemRequest) ([]m.Item, int64, error) {
		return nil, 0, errors.New("query failed")
	}

	resp, total, err := uc.Search(context.Background(), &dto.SearchItemRequest{Page: 1, Size: 10})

	assert.Error(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, resp)
}

// ---- Get tests ----

func TestItemUseCase_Get_Success(t *testing.T) {
	uc, _, repo := newTestItemUseCase(t)

	repo.findByIdFunc = func(_ context.Context, tx bun.IDB, id int64) (*m.Item, error) {
		assert.Nil(t, tx)
		assert.Equal(t, int64(1), id)
		return &m.Item{ID: 1, Name: "Widget"}, nil
	}

	resp, err := uc.Get(context.Background(), &dto.GetItemRequest{ID: 1})

	assert.NoError(t, err)
	assert.Equal(t, "Widget", resp.Name)
}

func TestItemUseCase_Get_NotFound(t *testing.T) {
	uc, _, repo := newTestItemUseCase(t)

	repo.findByIdFunc = func(_ context.Context, _ bun.IDB, _ int64) (*m.Item, error) {
		return nil, nil
	}

	resp, err := uc.Get(context.Background(), &dto.GetItemRequest{ID: 999})

	assert.Nil(t, resp)
	assert.Equal(t, apperror.ItemErrors.NotFound, err)
}

func TestItemUseCase_Get_RepositoryError(t *testing.T) {
	uc, _, repo := newTestItemUseCase(t)

	repo.findByIdFunc = func(_ context.Context, _ bun.IDB, _ int64) (*m.Item, error) {
		return nil, errors.New("db error")
	}

	resp, err := uc.Get(context.Background(), &dto.GetItemRequest{ID: 1})

	assert.Nil(t, resp)
	assert.Equal(t, apperror.ItemErrors.FailedToGet, err)
}

// ---- Update tests ----

func TestItemUseCase_Update_Success(t *testing.T) {
	uc, mock, repo := newTestItemUseCase(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	repo.findByIdFunc = func(_ context.Context, _ bun.IDB, id int64) (*m.Item, error) {
		return &m.Item{ID: id, Name: "Old", Stock: 5}, nil
	}
	repo.updateFunc = func(_ context.Context, _ bun.IDB, item *m.Item) error {
		assert.Equal(t, "New", item.Name)
		assert.Equal(t, int32(10), item.Stock)
		return nil
	}

	req := &dto.UpdateItemRequest{ID: 1, Name: "New", SKU: "SKU-1", Currency: "USD", Stock: 10}
	resp, err := uc.Update(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, "New", resp.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestItemUseCase_Update_InvalidRequest(t *testing.T) {
	uc, mock, _ := newTestItemUseCase(t)

	resp, err := uc.Update(context.Background(), &dto.UpdateItemRequest{})

	assert.Nil(t, resp)
	assert.Equal(t, apperror.ItemErrors.InvalidRequest, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestItemUseCase_Update_TransactionBeginError(t *testing.T) {
	uc, mock, _ := newTestItemUseCase(t)
	mock.ExpectBegin().WillReturnError(errors.New("connection refused"))

	resp, err := uc.Update(context.Background(), &dto.UpdateItemRequest{ID: 1, Stock: 1})

	assert.Nil(t, resp)
	assert.Equal(t, apperror.ItemErrors.FailedToUpdate, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestItemUseCase_Update_NotFound(t *testing.T) {
	uc, mock, repo := newTestItemUseCase(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	repo.findByIdFunc = func(_ context.Context, _ bun.IDB, _ int64) (*m.Item, error) {
		return nil, nil
	}

	resp, err := uc.Update(context.Background(), &dto.UpdateItemRequest{ID: 999, Stock: 1})

	assert.Nil(t, resp)
	assert.Equal(t, apperror.ItemErrors.NotFound, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestItemUseCase_Update_FindByIdError(t *testing.T) {
	uc, mock, repo := newTestItemUseCase(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	repo.findByIdFunc = func(_ context.Context, _ bun.IDB, _ int64) (*m.Item, error) {
		return nil, errors.New("db error")
	}

	resp, err := uc.Update(context.Background(), &dto.UpdateItemRequest{ID: 1, Stock: 1})

	assert.Nil(t, resp)
	assert.Equal(t, apperror.ItemErrors.FailedToUpdate, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestItemUseCase_Update_RepositoryError(t *testing.T) {
	uc, mock, repo := newTestItemUseCase(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	repo.findByIdFunc = func(_ context.Context, _ bun.IDB, id int64) (*m.Item, error) {
		return &m.Item{ID: id}, nil
	}
	repo.updateFunc = func(_ context.Context, _ bun.IDB, _ *m.Item) error {
		return errors.New("update failed")
	}

	resp, err := uc.Update(context.Background(), &dto.UpdateItemRequest{ID: 1, Stock: 1})

	assert.Nil(t, resp)
	assert.Equal(t, apperror.ItemErrors.FailedToUpdate, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---- Delete tests ----

func TestItemUseCase_Delete_Success(t *testing.T) {
	uc, mock, repo := newTestItemUseCase(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	repo.findByIdFunc = func(_ context.Context, _ bun.IDB, id int64) (*m.Item, error) {
		return &m.Item{ID: id}, nil
	}
	repo.deleteFunc = func(_ context.Context, _ bun.IDB, _ *m.Item) error {
		return nil
	}

	err := uc.Delete(context.Background(), &dto.DeleteItemRequest{ID: 1})

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestItemUseCase_Delete_TransactionBeginError(t *testing.T) {
	uc, mock, _ := newTestItemUseCase(t)
	mock.ExpectBegin().WillReturnError(errors.New("connection refused"))

	err := uc.Delete(context.Background(), &dto.DeleteItemRequest{ID: 1})

	assert.Equal(t, apperror.ItemErrors.FailedToDelete, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestItemUseCase_Delete_NotFound(t *testing.T) {
	uc, mock, repo := newTestItemUseCase(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	repo.findByIdFunc = func(_ context.Context, _ bun.IDB, _ int64) (*m.Item, error) {
		return nil, nil
	}

	err := uc.Delete(context.Background(), &dto.DeleteItemRequest{ID: 999})

	assert.Equal(t, apperror.ItemErrors.NotFound, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestItemUseCase_Delete_FindByIdError(t *testing.T) {
	uc, mock, repo := newTestItemUseCase(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	repo.findByIdFunc = func(_ context.Context, _ bun.IDB, _ int64) (*m.Item, error) {
		return nil, errors.New("db error")
	}

	err := uc.Delete(context.Background(), &dto.DeleteItemRequest{ID: 1})

	assert.Equal(t, apperror.ItemErrors.FailedToDelete, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestItemUseCase_Delete_RepositoryError(t *testing.T) {
	uc, mock, repo := newTestItemUseCase(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	repo.findByIdFunc = func(_ context.Context, _ bun.IDB, id int64) (*m.Item, error) {
		return &m.Item{ID: id}, nil
	}
	repo.deleteFunc = func(_ context.Context, _ bun.IDB, _ *m.Item) error {
		return errors.New("delete failed")
	}

	err := uc.Delete(context.Background(), &dto.DeleteItemRequest{ID: 1})

	assert.Equal(t, apperror.ItemErrors.FailedToDelete, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestItemUseCase_Delete_CommitError(t *testing.T) {
	uc, mock, repo := newTestItemUseCase(t)
	mock.ExpectBegin()
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	repo.findByIdFunc = func(_ context.Context, _ bun.IDB, id int64) (*m.Item, error) {
		return &m.Item{ID: id}, nil
	}
	repo.deleteFunc = func(_ context.Context, _ bun.IDB, _ *m.Item) error {
		return nil
	}

	err := uc.Delete(context.Background(), &dto.DeleteItemRequest{ID: 1})

	assert.Equal(t, apperror.ItemErrors.FailedToDelete, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

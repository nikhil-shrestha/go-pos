package repository

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/nikhil-shrestha/go-pos/internal/adapter/storage/postgres"
	"github.com/nikhil-shrestha/go-pos/internal/core/domain"
	"github.com/jackc/pgx/v5"
)

/**
 * PaymentRepository implements port.PaymentRepository interface
 * and provides an access to the postgres database
 */
type PaymentRepository struct {
	db *postgres.DB
}

// NewPaymentRepository creates a new payment repository instance
func NewPaymentRepository(db *postgres.DB) *PaymentRepository {
	return &PaymentRepository{
		db,
	}
}

// CreatePayment creates a new payment record in the database
func (pr *PaymentRepository) CreatePayment(ctx context.Context, payment *domain.Payment) (*domain.Payment, error) {
	query := pr.db.QueryBuilder.Insert("payments").
		Columns("name", "type", "logo").
		Values(payment.Name, payment.Type, payment.Logo).
		Suffix("RETURNING *")

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	err = pr.db.QueryRow(ctx, sql, args...).Scan(
		&payment.ID,
		&payment.Name,
		&payment.Type,
		&payment.Logo,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)
	if err != nil {
		if errCode := pr.db.ErrorCode(err); errCode == "23505" {
			return nil, domain.ErrConflictingData
		}
		return nil, err
	}

	return payment, nil
}

// GetPaymentByID retrieves a payment record from the database by id
func (pr *PaymentRepository) GetPaymentByID(ctx context.Context, id uint64) (*domain.Payment, error) {
	var payment domain.Payment

	query := pr.db.QueryBuilder.Select("*").
		From("payments").
		Where(sq.Eq{"id": id}).
		Limit(1)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	err = pr.db.QueryRow(ctx, sql, args...).Scan(
		&payment.ID,
		&payment.Name,
		&payment.Type,
		&payment.Logo,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrDataNotFound
		}
		return nil, err
	}

	return &payment, nil
}

// ListPayments retrieves a list of payments from the database
func (pr *PaymentRepository) ListPayments(ctx context.Context, skip, limit uint64) ([]domain.Payment, error) {
	var payment domain.Payment
	var payments []domain.Payment

	query := pr.db.QueryBuilder.Select("*").
		From("payments").
		OrderBy("id").
		Limit(limit).
		Offset((skip - 1) * limit)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := pr.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		err := rows.Scan(
			&payment.ID,
			&payment.Name,
			&payment.Type,
			&payment.Logo,
			&payment.CreatedAt,
			&payment.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		payments = append(payments, payment)
	}

	return payments, nil
}

// UpdatePayment updates a payment record in the database
func (pr *PaymentRepository) UpdatePayment(ctx context.Context, payment *domain.Payment) (*domain.Payment, error) {
	name := nullString(payment.Name)
	paymentType := nullString(string(payment.Type))
	logo := nullString(payment.Logo)

	query := pr.db.QueryBuilder.Update("payments").
		Set("name", sq.Expr("COALESCE(?, name)", name)).
		Set("type", sq.Expr("COALESCE(?, type)", paymentType)).
		Set("logo", sq.Expr("COALESCE(?, logo)", logo)).
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": payment.ID}).
		Suffix("RETURNING *")

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	err = pr.db.QueryRow(ctx, sql, args...).Scan(
		&payment.ID,
		&payment.Name,
		&payment.Type,
		&payment.Logo,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)
	if err != nil {
		if errCode := pr.db.ErrorCode(err); errCode == "23505" {
			return nil, domain.ErrConflictingData
		}
		return nil, err
	}

	return payment, nil
}

// DeletePayment deletes a payment record from the database by id
func (pr *PaymentRepository) DeletePayment(ctx context.Context, id uint64) error {
	query := pr.db.QueryBuilder.Delete("payments").
		Where(sq.Eq{"id": id})

	sql, args, err := query.ToSql()
	if err != nil {
		return err
	}

	_, err = pr.db.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}

	return nil
}

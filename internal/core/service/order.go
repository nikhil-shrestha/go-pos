package service

import (
	"context"

	"github.com/nikhil-shrestha/go-pos/internal/core/domain"
	"github.com/nikhil-shrestha/go-pos/internal/core/port"
	"github.com/nikhil-shrestha/go-pos/internal/core/util"
)

/**
 * OrderService implements port.OrderService, port.ProductService,
 * port.UserService and port.PaymentService interfaces and provides
 * an access to the order, product, user and payment repositories
 * and cache service
 */
type OrderService struct {
	orderRepo    port.OrderRepository
	productRepo  port.ProductRepository
	categoryRepo port.CategoryRepository
	userRepo     port.UserRepository
	paymentRepo  port.PaymentRepository
	cache        port.CacheRepository
}

// NewOrderService creates a new order service instance
func NewOrderService(orderRepo port.OrderRepository, productRepo port.ProductRepository, categoryRepo port.CategoryRepository, userRepo port.UserRepository, paymentRepo port.PaymentRepository, cache port.CacheRepository) *OrderService {
	return &OrderService{
		orderRepo,
		productRepo,
		categoryRepo,
		userRepo,
		paymentRepo,
		cache,
	}
}

// CreateOrder creates a new order
func (os *OrderService) CreateOrder(ctx context.Context, order *domain.Order) (*domain.Order, error) {
	var totalPrice float64
	for i, orderProduct := range order.Products {
		product, err := os.productRepo.GetProductByID(ctx, orderProduct.ProductID)
		if err != nil {
			if err == domain.ErrDataNotFound {
				return nil, err
			}
			return nil, domain.ErrInternal
		}

		if product.Stock < orderProduct.Quantity {
			return nil, domain.ErrInsufficientStock
		}

		order.Products[i].TotalPrice = product.Price * float64(orderProduct.Quantity)
		totalPrice += order.Products[i].TotalPrice
	}

	if order.TotalPaid < totalPrice {
		return nil, domain.ErrInsufficientPayment
	}

	order.TotalPrice = totalPrice
	order.TotalReturn = order.TotalPaid - order.TotalPrice

	order, err := os.orderRepo.CreateOrder(ctx, order)
	if err != nil {
		return nil, domain.ErrInternal
	}

	user, err := os.userRepo.GetUserByID(ctx, order.UserID)
	if err != nil {
		if err == domain.ErrDataNotFound {
			return nil, err
		}
		return nil, domain.ErrInternal
	}

	payment, err := os.paymentRepo.GetPaymentByID(ctx, order.PaymentID)
	if err != nil {
		if err == domain.ErrDataNotFound {
			return nil, err
		}
		return nil, domain.ErrInternal
	}

	order.User = user
	order.Payment = payment

	for i, orderProduct := range order.Products {
		product, err := os.productRepo.GetProductByID(ctx, orderProduct.ProductID)
		if err != nil {
			if err == domain.ErrDataNotFound {
				return nil, err
			}
			return nil, domain.ErrInternal
		}

		category, err := os.categoryRepo.GetCategoryByID(ctx, product.CategoryID)
		if err != nil {
			if err == domain.ErrDataNotFound {
				return nil, err
			}
			return nil, domain.ErrInternal
		}

		order.Products[i].Product = product
		order.Products[i].Product.Category = category
	}

	err = os.cache.DeleteByPrefix(ctx, "orders:*")
	if err != nil {
		return nil, domain.ErrInternal
	}

	cacheKey := util.GenerateCacheKey("order", order.ID)
	orderSerialized, err := util.Serialize(order)
	if err != nil {
		return nil, domain.ErrInternal
	}

	err = os.cache.Set(ctx, cacheKey, orderSerialized, 0)
	if err != nil {
		return nil, domain.ErrInternal
	}

	return order, nil
}

// GetOrder gets an order by ID
func (os *OrderService) GetOrder(ctx context.Context, id uint64) (*domain.Order, error) {
	var order *domain.Order

	cacheKey := util.GenerateCacheKey("order", id)
	cachedOrder, err := os.cache.Get(ctx, cacheKey)
	if err == nil {
		err := util.Deserialize(cachedOrder, &order)
		if err != nil {
			return nil, domain.ErrInternal
		}
		return order, nil
	}

	order, err = os.orderRepo.GetOrderByID(ctx, id)
	if err != nil {
		if err == domain.ErrDataNotFound {
			return nil, err
		}
		return nil, domain.ErrInternal
	}

	user, err := os.userRepo.GetUserByID(ctx, order.UserID)
	if err != nil {
		if err == domain.ErrDataNotFound {
			return nil, err
		}
		return nil, domain.ErrInternal
	}

	payment, err := os.paymentRepo.GetPaymentByID(ctx, order.PaymentID)
	if err != nil {
		if err == domain.ErrDataNotFound {
			return nil, err
		}
		return nil, domain.ErrInternal
	}

	order.User = user
	order.Payment = payment

	for i, orderProduct := range order.Products {
		product, err := os.productRepo.GetProductByID(ctx, orderProduct.ProductID)
		if err != nil {
			if err == domain.ErrDataNotFound {
				return nil, err
			}
			return nil, domain.ErrInternal
		}

		category, err := os.categoryRepo.GetCategoryByID(ctx, product.CategoryID)
		if err != nil {
			if err == domain.ErrDataNotFound {
				return nil, err
			}
			return nil, domain.ErrInternal
		}

		order.Products[i].Product = product
		order.Products[i].Product.Category = category
	}

	orderSerialized, err := util.Serialize(order)
	if err != nil {
		return nil, domain.ErrInternal
	}

	err = os.cache.Set(ctx, cacheKey, orderSerialized, 0)
	if err != nil {
		return nil, domain.ErrInternal
	}

	return order, nil
}

// ListOrders lists all orders
func (os *OrderService) ListOrders(ctx context.Context, skip, limit uint64) ([]domain.Order, error) {
	var orders []domain.Order

	params := util.GenerateCacheKeyParams(skip, limit)
	cacheKey := util.GenerateCacheKey("orders", params)

	cachedOrders, err := os.cache.Get(ctx, cacheKey)
	if err == nil {
		err := util.Deserialize(cachedOrders, &orders)
		if err != nil {
			return nil, domain.ErrInternal
		}
		return orders, nil
	}

	orders, err = os.orderRepo.ListOrders(ctx, skip, limit)
	if err != nil {
		return nil, domain.ErrInternal
	}

	for i, order := range orders {
		user, err := os.userRepo.GetUserByID(ctx, order.UserID)
		if err != nil {
			if err == domain.ErrDataNotFound {
				return nil, err
			}
			return nil, domain.ErrInternal
		}

		payment, err := os.paymentRepo.GetPaymentByID(ctx, order.PaymentID)
		if err != nil {
			if err == domain.ErrDataNotFound {
				return nil, err
			}
			return nil, domain.ErrInternal
		}

		orders[i].User = user
		orders[i].Payment = payment
	}

	for i, order := range orders {
		for j, orderProduct := range order.Products {
			product, err := os.productRepo.GetProductByID(ctx, orderProduct.ProductID)
			if err != nil {
				if err == domain.ErrDataNotFound {
					return nil, err
				}
				return nil, domain.ErrInternal
			}

			category, err := os.categoryRepo.GetCategoryByID(ctx, product.CategoryID)
			if err != nil {
				if err == domain.ErrDataNotFound {
					return nil, err
				}
				return nil, domain.ErrInternal
			}

			orders[i].Products[j].Product = product
			orders[i].Products[j].Product.Category = category
		}
	}

	ordersSerialized, err := util.Serialize(orders)
	if err != nil {
		return nil, domain.ErrInternal
	}

	err = os.cache.Set(ctx, cacheKey, ordersSerialized, 0)
	if err != nil {
		return nil, domain.ErrInternal
	}

	return orders, nil
}

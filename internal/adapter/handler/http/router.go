package http

import (
	"log/slog"
	"strings"

	"github.com/nikhil-shrestha/go-pos/internal/adapter/config"
	"github.com/nikhil-shrestha/go-pos/internal/core/port"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	sloggin "github.com/samber/slog-gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Router is a wrapper for HTTP router
type Router struct {
	*gin.Engine
}

// NewRouter creates a new HTTP router
func NewRouter(
	config *config.HTTP,
	token port.TokenService,
	userHandler UserHandler,
	authHandler AuthHandler,
	paymentHandler PaymentHandler,
	categoryHandler CategoryHandler,
	productHandler ProductHandler,
	orderHandler OrderHandler,
) (*Router, error) {
	// Disable debug mode in production
	if config.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// CORS
	ginConfig := cors.DefaultConfig()
	allowedOrigins := config.AllowedOrigins
	originsList := strings.Split(allowedOrigins, ",")
	ginConfig.AllowOrigins = originsList

	router := gin.New()
	router.Use(sloggin.New(slog.Default()), gin.Recovery(), cors.New(ginConfig))

	// Custom validators
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if ok {
		if err := v.RegisterValidation("user_role", userRoleValidator); err != nil {
			return nil, err
		}

		if err := v.RegisterValidation("payment_type", paymentTypeValidator); err != nil {
			return nil, err
		}

	}

	// Swagger
	router.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := router.Group("/v1")
	{
		user := v1.Group("/users")
		{
			user.POST("/", userHandler.Register)
			user.POST("/login", authHandler.Login)

			authUser := user.Group("/").Use(authMiddleware(token))
			{
				authUser.GET("/", userHandler.ListUsers)
				authUser.GET("/:id", userHandler.GetUser)

				admin := authUser.Use(adminMiddleware())
				{
					admin.PUT("/:id", userHandler.UpdateUser)
					admin.DELETE("/:id", userHandler.DeleteUser)
				}
			}
		}
		payment := v1.Group("/payments").Use(authMiddleware(token))
		{
			payment.GET("/", paymentHandler.ListPayments)
			payment.GET("/:id", paymentHandler.GetPayment)

			admin := payment.Use(adminMiddleware())
			{
				admin.POST("/", paymentHandler.CreatePayment)
				admin.PUT("/:id", paymentHandler.UpdatePayment)
				admin.DELETE("/:id", paymentHandler.DeletePayment)
			}
		}
		category := v1.Group("/categories").Use(authMiddleware(token))
		{
			category.GET("/", categoryHandler.ListCategories)
			category.GET("/:id", categoryHandler.GetCategory)

			admin := category.Use(adminMiddleware())
			{
				admin.POST("/", categoryHandler.CreateCategory)
				admin.PUT("/:id", categoryHandler.UpdateCategory)
				admin.DELETE("/:id", categoryHandler.DeleteCategory)
			}
		}
		product := v1.Group("/products").Use(authMiddleware(token))
		{
			product.GET("/", productHandler.ListProducts)
			product.GET("/:id", productHandler.GetProduct)

			admin := product.Use(adminMiddleware())
			{
				admin.POST("/", productHandler.CreateProduct)
				admin.PUT("/:id", productHandler.UpdateProduct)
				admin.DELETE("/:id", productHandler.DeleteProduct)
			}
		}
		order := v1.Group("/orders").Use(authMiddleware(token))
		{
			order.POST("/", orderHandler.CreateOrder)
			order.GET("/", orderHandler.ListOrders)
			order.GET("/:id", orderHandler.GetOrder)
		}
	}

	return &Router{
		router,
	}, nil
}

// Serve starts the HTTP server
func (r *Router) Serve(listenAddr string) error {
	return r.Run(listenAddr)
}

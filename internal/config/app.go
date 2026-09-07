package config

import (
	"golang-clean-architecture/internal/delivery/http"
	"golang-clean-architecture/internal/delivery/http/middleware"
	"golang-clean-architecture/internal/delivery/http/route"
	"golang-clean-architecture/internal/repository"
	"golang-clean-architecture/internal/usecase"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/uptrace/bun"
)

type BootstrapConfig struct {
	DB       *bun.DB
	App      *echo.Echo
	Log      *logrus.Logger
	Validate *validator.Validate
	Config   *viper.Viper
}

func Bootstrap(config *BootstrapConfig) {
	// setup repositories
	userRepository := repository.NewUserRepository(config.DB, config.Log)
	contactRepository := repository.NewContactRepository(config.DB, config.Log)
	addressRepository := repository.NewAddressRepository(config.DB, config.Log)
	itemRepository := repository.NewItemRepository(config.DB, config.Log)

	jwtSecret := config.Config.GetString("jwt.secret")
	jwtTTLMinutes := config.Config.GetInt("jwt.ttl_minutes")

	// setup use cases
	userUseCase := usecase.NewUserUseCase(
		config.DB,
		config.Log,
		config.Validate,
		userRepository,
		jwtSecret,
		time.Duration(jwtTTLMinutes)*time.Minute,
	)
	contactUseCase := usecase.NewContactUseCase(config.DB, config.Log, config.Validate, contactRepository)
	addressUseCase := usecase.NewAddressUseCase(config.DB, config.Log, config.Validate, contactRepository, addressRepository)
	itemUseCase := usecase.NewItemUseCase(config.DB, config.Log, config.Validate, itemRepository)

	// setup controller
	userController := http.NewUserController(userUseCase, config.Log)
	contactController := http.NewContactController(contactUseCase, config.Log)
	addressController := http.NewAddressController(addressUseCase, config.Log)
	itemController := http.NewItemController(itemUseCase, config.Log)

	// setup middleware
	authMiddleware := middleware.NewAuth(jwtSecret)

	routeConfig := route.RouteConfig{
		App:               config.App,
		UserController:    userController,
		ContactController: contactController,
		AddressController: addressController,
		AuthMiddleware:    authMiddleware,
		ItemController:    itemController,
	}
	routeConfig.Setup()
}

package shop

import (
	"context"
	"log/slog"

	"github.com/WilliamTrojniak/TabAppBackend/db"
	"github.com/WilliamTrojniak/TabAppBackend/models"
	"github.com/WilliamTrojniak/TabAppBackend/services"
	"github.com/WilliamTrojniak/TabAppBackend/services/authorization"
	"github.com/WilliamTrojniak/TabAppBackend/services/sessions"
)

type Handler struct {
	logger      *slog.Logger
	store       *db.PgxStore
	sessions    *sessions.Handler
	handleError services.HTTPErrorHandler
}

func NewHandler(store *db.PgxStore, sessions *sessions.Handler, handleError services.HTTPErrorHandler, logger *slog.Logger) *Handler {
	return &Handler{
		logger:      logger,
		sessions:    sessions,
		store:       store,
		handleError: handleError,
	}
}

func (h *Handler) CreateShop(ctx context.Context, session *sessions.AuthedSession, data *models.ShopCreate) (shop *models.Shop, err error) {
	h.logger.Debug("Creating shop")

	// Data validation
	err = models.ValidateData(data, h.logger)
	if err != nil {
		return nil, err
	}

	return db.WithTxRet(ctx, h.store, func(pq *db.PgxQueries) (*models.Shop, error) {
		user, err := pq.GetUser(ctx, session.UserId)
		if err != nil {
			return nil, err
		}

		targetUser, err := pq.GetUser(ctx, data.OwnerId)
		if err != nil {
			return nil, err
		}

		if ok, err := authorization.AuthorizeUserAction(user, targetUser, authorization.USER_ACTION_CREATE_SHOP); err != nil {
			return nil, err
		} else if !ok {
			return nil, services.NewUnauthorizedServiceError(nil)
		}

		shopId, err := pq.CreateShop(ctx, data)
		if err != nil {
			return nil, err
		}

		shop, err := pq.GetShopById(ctx, shopId)
		if err != nil {
			return nil, err
		}

		return shop, nil
	})
}

func (h *Handler) GetShops(ctx context.Context, params *models.GetShopsQueryParams) ([]models.ShopOverview, error) {
	if params == nil {
		params = &models.GetShopsQueryParams{
			Offset: 0,
			Limit:  10,
		}
	}

	return db.WithTxRet(ctx, h.store, func(pq *db.PgxQueries) ([]models.ShopOverview, error) {
		shops, err := pq.GetShops(ctx, params)
		if err != nil {
			h.logger.Warn("Error reading from database", "error", err)
			return nil, err
		}
		return shops, nil
	})
}

func (h *Handler) GetShopById(ctx context.Context, session *sessions.AuthedSession, shopId int) (shop *models.Shop, err error) {
	err = WithAuthorizeShopAction(ctx, h.store, session, shopId, authorization.SHOP_ACTION_READ, func(pq *db.PgxQueries, user *models.User, s *models.Shop) error {
		shop = s
		return nil
	})
	if err != nil {
		return nil, err
	}

	return shop, nil
}

func (h *Handler) UpdateShop(ctx context.Context, session *sessions.AuthedSession, shopId int, data *models.ShopUpdate) error {
	h.logger.Debug("Updating Shop", "id", shopId)

	err := models.ValidateData(data, h.logger)
	if err != nil {
		return err
	}

	return WithAuthorizeShopAction(ctx, h.store, session, shopId, authorization.SHOP_ACTION_UPDATE, func(pq *db.PgxQueries, _ *models.User, _ *models.Shop) error {
		return pq.UpdateShop(ctx, shopId, data)
	})
}

func (h *Handler) DeleteShop(ctx context.Context, session *sessions.AuthedSession, shopId int) error {
	return WithAuthorizeShopAction(ctx, h.store, session, shopId, authorization.SHOP_ACTION_DELETE, func(pq *db.PgxQueries, _ *models.User, _ *models.Shop) error {
		return pq.DeleteShop(ctx, shopId)
	})
}

func WithAuthorizeShopAction(ctx context.Context, conn db.PgxConn, session *sessions.AuthedSession, shopId int, action authorization.Action, fn func(pq *db.PgxQueries, user *models.User, shop *models.Shop) error) error {
	return db.WithTx(ctx, conn, func(pq *db.PgxQueries) error {
		user, err := pq.GetUser(ctx, session.UserId)
		if err != nil {
			return err
		}

		shop, err := pq.GetShopById(ctx, shopId)
		if err != nil {
			return err
		}

		if ok, err := authorization.AuthorizeShopAction(user, shop, action); err != nil {
			return err
		} else if !ok {
			return services.NewUnauthorizedServiceError(nil)
		}
		return fn(pq, user, shop)
	})
}

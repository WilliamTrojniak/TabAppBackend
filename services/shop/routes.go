package shop

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/WilliamTrojniak/TabAppBackend/services"
	"github.com/WilliamTrojniak/TabAppBackend/types"
)

const (
	shopIdParam              = "shopId"
	categoryIdParam          = "categoryId"
	itemIdParam              = "itemId"
	itemVariantIdParam       = "itemVariantId"
	substitutionGroupIdParam = "substitutionGroupId"
	tabIdParam               = "tabId"
	billIdParam              = "billId"
)

func (h *Handler) RegisterRoutes(router *http.ServeMux) {
	h.logger.Info("Registering shop routes")
	router.HandleFunc("POST /shops", h.handleCreateShop)
	router.HandleFunc("GET /shops", h.handleGetShops)

	// Payment Methods
	router.HandleFunc("GET /payment-methods", h.handleGetPaymentMethods)

	// Shops
	router.HandleFunc(fmt.Sprintf("GET /shops/{%v}", shopIdParam), h.handleGetShopById)
	router.HandleFunc(fmt.Sprintf("PATCH /shops/{%v}", shopIdParam), h.handleUpdateShop)
	router.HandleFunc(fmt.Sprintf("DELETE /shops/{%v}", shopIdParam), h.handleDeleteShop)

	// Categories
	router.HandleFunc(fmt.Sprintf("POST /shops/{%v}/categories", shopIdParam), h.handleCreateCategory)
	router.HandleFunc(fmt.Sprintf("GET /shops/{%v}/categories", shopIdParam), h.handleGetCategories)
	router.HandleFunc(fmt.Sprintf("PATCH /shops/{%v}/categories/{%v}", shopIdParam, categoryIdParam), h.handleUpdateCategory)
	router.HandleFunc(fmt.Sprintf("DELETE /shops/{%v}/categories/{%v}", shopIdParam, categoryIdParam), h.handleDeleteCategory)

	// Items
	router.HandleFunc(fmt.Sprintf("POST /shops/{%v}/items", shopIdParam), h.handleCreateItem)
	router.HandleFunc(fmt.Sprintf("GET /shops/{%v}/items", shopIdParam), h.handleGetItems)
	router.HandleFunc(fmt.Sprintf("PATCH /shops/{%v}/items/{%v}", shopIdParam, itemIdParam), h.handleUpdateItem)
	router.HandleFunc(fmt.Sprintf("GET /shops/{%v}/items/{%v}", shopIdParam, itemIdParam), h.handleGetItem)
	router.HandleFunc(fmt.Sprintf("DELETE /shops/{%v}/items/{%v}", shopIdParam, itemIdParam), h.handleDeleteItem)

	// Item Variants
	router.HandleFunc(fmt.Sprintf("POST /shops/{%v}/items/{%v}/variants", shopIdParam, itemIdParam), h.handleCreateItemVariant)
	router.HandleFunc(fmt.Sprintf("PATCH /shops/{%v}/items/{%v}/variants/{%v}", shopIdParam, itemIdParam, itemVariantIdParam), h.handleUpdateItemVariant)
	router.HandleFunc(fmt.Sprintf("DELETE /shops/{%v}/items/{%v}/variants/{%v}", shopIdParam, itemIdParam, itemVariantIdParam), h.handleDeleteItemVariant)

	// Item Substitution Groups
	router.HandleFunc(fmt.Sprintf("POST /shops/{%v}/substitutions", shopIdParam), h.handleCreateSubstitutionGroup)
	router.HandleFunc(fmt.Sprintf("GET /shops/{%v}/substitutions", shopIdParam), h.handleGetSubstitutionGroups)
	router.HandleFunc(fmt.Sprintf("PATCH /shops/{%v}/substitutions/{%v}", shopIdParam, substitutionGroupIdParam), h.handleUpdateSubstitutionGroup)
	router.HandleFunc(fmt.Sprintf("DELETE /shops/{%v}/substitutions/{%v}", shopIdParam, substitutionGroupIdParam), h.handleDeleteSubstitutionGroup)

	// Tabs
	router.HandleFunc(fmt.Sprintf("POST /shops/{%v}/tabs", shopIdParam), h.handleCreateTab)
	router.HandleFunc(fmt.Sprintf("GET /shops/{%v}/tabs", shopIdParam), h.handleGetTabs)
	router.HandleFunc(fmt.Sprintf("GET /shops/{%v}/tabs/{%v}", shopIdParam, tabIdParam), h.handleGetTabById)
	router.HandleFunc(fmt.Sprintf("PATCH /shops/{%v}/tabs/{%v}", shopIdParam, tabIdParam), h.handleUpdateTab)
	router.HandleFunc(fmt.Sprintf("POST /shops/{%v}/tabs/{%v}/approve", shopIdParam, tabIdParam), h.handleApproveTab)
	router.HandleFunc(fmt.Sprintf("POST /shops/{%v}/tabs/{%v}/bills/{%v}/close", shopIdParam, tabIdParam, billIdParam), h.handleCloseTabBill)

	// Orders
	router.HandleFunc(fmt.Sprintf("POST /shops/{%v}/tabs/{%v}/add-order", shopIdParam, tabIdParam), h.handleAddOrderToTab)
	router.HandleFunc(fmt.Sprintf("POST /shops/{%v}/tabs/{%v}/remove-order", shopIdParam, tabIdParam), h.handleRemoveOrderFromTab)

}

func (h *Handler) handleCreateShop(w http.ResponseWriter, r *http.Request) {
	session, err := h.sessions.GetSession(r)
	if err != nil {
		h.handleError(w, err)
		return
	}

	data := &types.ShopCreate{}
	err = types.ReadRequestJson(r, data)
	if err != nil {
		h.handleError(w, err)
		return
	}

	if data.OwnerId == "" {
		userId, err := session.GetUserId()
		if err != nil {
			h.handleError(w, err)
			return
		}
		data.OwnerId = userId
	}

	err = h.CreateShop(r.Context(), session, data)
	if err != nil {
		h.handleError(w, err)
		return
	}

}

func (h *Handler) handleGetShops(w http.ResponseWriter, r *http.Request) {
	// Query params
	userIdKey := "userId"

	searchParams := r.URL.Query()
	if searchParams.Has(userIdKey) {
		userId := searchParams.Get(userIdKey)
		shops, err := h.GetShopsByUserId(r.Context(), userId)
		if err != nil {
			h.handleError(w, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(shops)
		return
	}

	// TODO: Dynamically change limit and offset
	shops, err := h.GetShops(r.Context(), 10, 0)
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(shops)
	return
}

func (h *Handler) handleGetShopById(w http.ResponseWriter, r *http.Request) {
	shopId, err := strconv.Atoi(r.PathValue(shopIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shopId"))
		return
	}

	shop, err := h.GetShopById(r.Context(), shopId)
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(shop)

}

func (h *Handler) handleUpdateShop(w http.ResponseWriter, r *http.Request) {
	shopId, err := strconv.Atoi(r.PathValue(shopIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shopId"))
		return
	}

	session, err := h.sessions.GetSession(r)
	if err != nil {
		h.handleError(w, err)
		return
	}

	data := types.ShopUpdate{}
	err = types.ReadRequestJson(r, &data)
	if err != nil {
		h.handleError(w, err)
		return
	}

	err = h.UpdateShop(r.Context(), session, shopId, &data)
	if err != nil {
		h.handleError(w, err)
		return
	}
}

func (h *Handler) handleDeleteShop(w http.ResponseWriter, r *http.Request) {
	shopId, err := strconv.Atoi(r.PathValue(shopIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shopId"))
		return
	}

	session, err := h.sessions.GetSession(r)
	if err != nil {
		h.handleError(w, err)
		return
	}

	err = h.DeleteShop(r.Context(), session, shopId)
	if err != nil {
		h.handleError(w, err)
		return
	}
}

func (h *Handler) handleGetPaymentMethods(w http.ResponseWriter, r *http.Request) {
	methods := make([]types.PaymentMethod, 0)
	methods = append(methods, types.PaymentMethodInPerson, types.PaymentMethodChartstring)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(methods)
}

func (h *Handler) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	session, err := h.sessions.GetSession(r)
	if err != nil {
		h.handleError(w, err)
		return
	}

	shopId, err := strconv.Atoi(r.PathValue(shopIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shop id"))
		return
	}

	data := types.CategoryCreate{}
	err = types.ReadRequestJson(r, &data)
	if err != nil {
		h.handleError(w, err)
		return
	}
	data.ShopId = shopId

	err = h.CreateCategory(r.Context(), session, &data)
	if err != nil {
		h.handleError(w, err)
		return
	}
}

func (h *Handler) handleGetCategories(w http.ResponseWriter, r *http.Request) {
	shopId, err := strconv.Atoi(r.PathValue(shopIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shop id"))
		return
	}

	categories, err := h.GetCategories(r.Context(), shopId)
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categories)
}

func (h *Handler) handleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	session, err := h.sessions.GetSession(r)
	if err != nil {
		h.handleError(w, err)
		return
	}

	shopId, err := strconv.Atoi(r.PathValue(shopIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shop id"))
		return
	}

	categoryId, err := strconv.Atoi(r.PathValue(categoryIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shop id"))
		return
	}

	data := types.CategoryUpdate{}
	err = types.ReadRequestJson(r, &data)
	if err != nil {
		h.handleError(w, err)
		return
	}

	err = h.UpdateCategory(r.Context(), session, shopId, categoryId, &data)
	if err != nil {
		h.handleError(w, err)
		return
	}
}

func (h *Handler) handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	session, err := h.sessions.GetSession(r)
	if err != nil {
		h.handleError(w, err)
		return
	}

	shopId, err := strconv.Atoi(r.PathValue(shopIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shop id"))
		return
	}

	categoryId, err := strconv.Atoi(r.PathValue(categoryIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shop id"))
		return
	}

	err = h.DeleteCategory(r.Context(), session, shopId, categoryId)
	if err != nil {
		h.handleError(w, err)
		return
	}

}

func (h *Handler) handleCreateItem(w http.ResponseWriter, r *http.Request) {
	session, err := h.sessions.GetSession(r)
	if err != nil {
		h.handleError(w, err)
		return
	}

	shopId, err := strconv.Atoi(r.PathValue(shopIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shop id"))
		return
	}

	data := types.ItemCreate{}
	err = types.ReadRequestJson(r, &data)
	if err != nil {
		h.handleError(w, err)
		return
	}
	data.ShopId = shopId

	err = h.CreateItem(r.Context(), session, &data)
	if err != nil {
		h.handleError(w, err)
		return
	}
}

func (h *Handler) handleGetItems(w http.ResponseWriter, r *http.Request) {
	shopId, err := strconv.Atoi(r.PathValue(shopIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shop id"))
		return
	}

	items, err := h.GetItems(r.Context(), shopId)
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (h *Handler) handleUpdateItem(w http.ResponseWriter, r *http.Request) {
	session, err := h.sessions.GetSession(r)
	if err != nil {
		h.handleError(w, err)
		return
	}

	shopId, err := strconv.Atoi(r.PathValue(shopIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shop id"))
		return
	}

	itemId, err := strconv.Atoi(r.PathValue(itemIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid item id"))
		return
	}

	data := types.ItemUpdate{}
	err = types.ReadRequestJson(r, &data)
	if err != nil {
		h.handleError(w, err)
		return
	}

	err = h.UpdateItem(r.Context(), session, shopId, itemId, &data)
	if err != nil {
		h.handleError(w, err)
		return
	}

}

func (h *Handler) handleGetItem(w http.ResponseWriter, r *http.Request) {
	shopId, err := strconv.Atoi(r.PathValue(shopIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shop id"))
		return
	}

	itemId, err := strconv.Atoi(r.PathValue(itemIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid item id"))
		return
	}

	item, err := h.GetItem(r.Context(), shopId, itemId)
	if err != nil {
		h.handleError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

func (h *Handler) handleDeleteItem(w http.ResponseWriter, r *http.Request) {
	session, err := h.sessions.GetSession(r)
	if err != nil {
		h.handleError(w, err)
		return
	}

	shopId, err := strconv.Atoi(r.PathValue(shopIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shop id"))
		return
	}

	itemId, err := strconv.Atoi(r.PathValue(itemIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid item id"))
		return
	}

	err = h.DeleteItem(r.Context(), session, shopId, itemId)
	if err != nil {
		h.handleError(w, err)
		return
	}

}

func (h *Handler) handleCreateItemVariant(w http.ResponseWriter, r *http.Request) {
	session, err := h.sessions.GetSession(r)
	if err != nil {
		h.handleError(w, err)
		return
	}

	shopId, err := strconv.Atoi(r.PathValue(shopIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shop id"))
		return
	}

	itemId, err := strconv.Atoi(r.PathValue(itemIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid item id"))
		return
	}

	data := types.ItemVariantCreate{}
	err = types.ReadRequestJson(r, &data)
	if err != nil {
		h.handleError(w, err)
		return
	}
	data.ShopId = shopId
	data.ItemId = itemId

	err = h.CreateItemVariant(r.Context(), session, &data)
	if err != nil {
		h.handleError(w, err)
		return
	}

	return
}

func (h *Handler) handleUpdateItemVariant(w http.ResponseWriter, r *http.Request) {
	session, err := h.sessions.GetSession(r)
	if err != nil {
		h.handleError(w, err)
		return
	}

	shopId, err := strconv.Atoi(r.PathValue(shopIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shop id"))
		return
	}

	itemId, err := strconv.Atoi(r.PathValue(itemIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid item id"))
		return
	}
	variantId, err := strconv.Atoi(r.PathValue(itemVariantIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid item variant id"))
		return
	}

	data := types.ItemVariantUpdate{}
	err = types.ReadRequestJson(r, &data)
	if err != nil {
		h.handleError(w, err)
		return
	}

	err = h.UpdateItemVariant(r.Context(), session, shopId, itemId, variantId, &data)
	if err != nil {
		h.handleError(w, err)
		return
	}
}

func (h *Handler) handleDeleteItemVariant(w http.ResponseWriter, r *http.Request) {
	session, err := h.sessions.GetSession(r)
	if err != nil {
		h.handleError(w, err)
		return
	}

	shopId, err := strconv.Atoi(r.PathValue(shopIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shop id"))
		return
	}

	itemId, err := strconv.Atoi(r.PathValue(itemIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid item id"))
		return
	}
	variantId, err := strconv.Atoi(r.PathValue(itemVariantIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid item variant id"))
		return
	}

	err = h.DeleteItemVariant(r.Context(), session, shopId, itemId, variantId)
	if err != nil {
		h.handleError(w, err)
		return
	}
}

func (h *Handler) handleCreateSubstitutionGroup(w http.ResponseWriter, r *http.Request) {
	session, err := h.sessions.GetSession(r)
	if err != nil {
		h.handleError(w, err)
		return
	}

	shopId, err := strconv.Atoi(r.PathValue(shopIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shop id"))
		return
	}

	data := types.SubstitutionGroupCreate{}
	err = types.ReadRequestJson(r, &data)
	if err != nil {
		h.handleError(w, err)
		return
	}
	data.ShopId = shopId

	err = h.CreateSubstitutionGroup(r.Context(), session, &data)
	if err != nil {
		h.handleError(w, err)
		return
	}
}

func (h *Handler) handleUpdateSubstitutionGroup(w http.ResponseWriter, r *http.Request) {
	session, err := h.sessions.GetSession(r)
	if err != nil {
		h.handleError(w, err)
		return
	}

	shopId, err := strconv.Atoi(r.PathValue(shopIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shop id"))
		return
	}

	substitutionGroupId, err := strconv.Atoi(r.PathValue(substitutionGroupIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid substitution id"))
		return
	}

	data := types.SubstitutionGroupUpdate{}
	err = types.ReadRequestJson(r, &data)
	if err != nil {
		h.handleError(w, err)
		return
	}

	err = h.UpdateSubstitutionGroup(r.Context(), session, shopId, substitutionGroupId, &data)
	if err != nil {
		h.handleError(w, err)
		return
	}
}

func (h *Handler) handleGetSubstitutionGroups(w http.ResponseWriter, r *http.Request) {
	shopId, err := strconv.Atoi(r.PathValue(shopIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shop id"))
		return
	}

	substitutionGroups, err := h.GetSubstitutionGroups(r.Context(), shopId)
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(substitutionGroups)
}

func (h *Handler) handleDeleteSubstitutionGroup(w http.ResponseWriter, r *http.Request) {
	session, err := h.sessions.GetSession(r)
	if err != nil {
		h.handleError(w, err)
		return
	}

	shopId, err := strconv.Atoi(r.PathValue(shopIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shop id"))
		return
	}

	substitutionGroupId, err := strconv.Atoi(r.PathValue(substitutionGroupIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid substitution id"))
		return
	}

	err = h.DeleteSubstitutionGroup(r.Context(), session, shopId, substitutionGroupId)
	if err != nil {
		h.handleError(w, err)
		return
	}

}

func (h *Handler) handleCreateTab(w http.ResponseWriter, r *http.Request) {
	session, err := h.sessions.GetSession(r)
	if err != nil {
		h.handleError(w, err)
		return
	}
	userId, err := session.GetUserId()
	if err != nil {
		h.handleError(w, err)
		return
	}

	shopId, err := strconv.Atoi(r.PathValue(shopIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shop id"))
		return
	}

	data := types.TabCreate{}
	err = types.ReadRequestJson(r, &data)
	if err != nil {
		h.handleError(w, err)
		return
	}
	data.ShopId = shopId
	data.OwnerId = userId

	err = h.CreateTab(r.Context(), session, &data)
	if err != nil {
		h.handleError(w, err)
		return
	}
}

func (h *Handler) handleGetTabs(w http.ResponseWriter, r *http.Request) {
	session, err := h.sessions.GetSession(r)
	if err != nil {
		h.handleError(w, err)
		return
	}

	shopId, err := strconv.Atoi(r.PathValue(shopIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shop id"))
		return
	}

	tabs, err := h.GetTabs(r.Context(), session, shopId)
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tabs)

}

func (h *Handler) handleGetTabById(w http.ResponseWriter, r *http.Request) {
	session, err := h.sessions.GetSession(r)
	if err != nil {
		h.handleError(w, err)
		return
	}

	shopId, err := strconv.Atoi(r.PathValue(shopIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shop id"))
		return
	}
	tabId, err := strconv.Atoi(r.PathValue(tabIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid tab id"))
		return
	}

	tab, err := h.GetTabById(r.Context(), session, shopId, tabId)
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tab)

}

func (h *Handler) handleUpdateTab(w http.ResponseWriter, r *http.Request) {
	session, err := h.sessions.GetSession(r)
	if err != nil {
		h.handleError(w, err)
		return
	}

	shopId, err := strconv.Atoi(r.PathValue(shopIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shop id"))
		return
	}

	tabId, err := strconv.Atoi(r.PathValue(tabIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid tab id"))
		return
	}

	data := types.TabUpdate{}
	err = types.ReadRequestJson(r, &data)
	if err != nil {
		h.handleError(w, err)
		return
	}

	err = h.UpdateTab(r.Context(), session, shopId, tabId, &data)
	if err != nil {
		h.handleError(w, err)
		return
	}

}

func (h *Handler) handleApproveTab(w http.ResponseWriter, r *http.Request) {
	session, err := h.sessions.GetSession(r)
	if err != nil {
		h.handleError(w, err)
		return
	}

	shopId, err := strconv.Atoi(r.PathValue(shopIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shop id"))
		return
	}

	tabId, err := strconv.Atoi(r.PathValue(tabIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid tab id"))
		return
	}

	err = h.ApproveTab(r.Context(), session, shopId, tabId)
	if err != nil {
		h.handleError(w, err)
		return
	}
}

func (h *Handler) handleAddOrderToTab(w http.ResponseWriter, r *http.Request) {
	session, err := h.sessions.GetSession(r)
	if err != nil {
		h.handleError(w, err)
		return
	}

	shopId, err := strconv.Atoi(r.PathValue(shopIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shop id"))
		return
	}

	tabId, err := strconv.Atoi(r.PathValue(tabIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid tab id"))
		return
	}

	data := types.BillOrderCreate{}
	err = types.ReadRequestJson(r, &data)
	if err != nil {
		h.handleError(w, err)
		return
	}

	err = h.AddOrderToTab(r.Context(), session, shopId, tabId, &data)
	if err != nil {
		h.handleError(w, err)
		return
	}

}

func (h *Handler) handleRemoveOrderFromTab(w http.ResponseWriter, r *http.Request) {
	session, err := h.sessions.GetSession(r)
	if err != nil {
		h.handleError(w, err)
		return
	}

	shopId, err := strconv.Atoi(r.PathValue(shopIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shop id"))
		return
	}

	tabId, err := strconv.Atoi(r.PathValue(tabIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid tab id"))
		return
	}

	data := types.BillOrderCreate{}
	err = types.ReadRequestJson(r, &data)
	if err != nil {
		h.handleError(w, err)
		return
	}

	err = h.RemoveOrderFromTab(r.Context(), session, shopId, tabId, &data)
	if err != nil {
		h.handleError(w, err)
		return
	}

}

func (h *Handler) handleCloseTabBill(w http.ResponseWriter, r *http.Request) {
	session, err := h.sessions.GetSession(r)
	if err != nil {
		h.handleError(w, err)
		return
	}

	shopId, err := strconv.Atoi(r.PathValue(shopIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid shop id"))
		return
	}

	tabId, err := strconv.Atoi(r.PathValue(tabIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid tab id"))
		return
	}

	billId, err := strconv.Atoi(r.PathValue(billIdParam))
	if err != nil {
		h.handleError(w, services.NewValidationServiceError(err, "Invalid bill id"))
		return
	}

	err = h.MarkTabBillPaid(r.Context(), session, shopId, tabId, billId)
	if err != nil {
		h.handleError(w, err)
		return
	}

}

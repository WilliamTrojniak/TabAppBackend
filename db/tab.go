package db

import (
	"context"
	"errors"

	"github.com/WilliamTrojniak/TabAppBackend/services"
	"github.com/WilliamTrojniak/TabAppBackend/types"
	"github.com/jackc/pgx/v5"
)

func (s *PgxStore) CreateTab(ctx context.Context, data *types.TabCreate) error {

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return handlePgxError(err)
	}

	defer tx.Rollback(ctx)
	row := tx.QueryRow(ctx, `
    INSERT INTO tabs 
      (shop_id, owner_id, payment_method, organization, display_name,
      start_date, end_date, daily_start_time, daily_end_time, active_days_of_wk,
      dollar_limit_per_order, verification_method, payment_details, billing_interval_days) 
    VALUES (@shopId, @ownerId, @paymentMethod, @organization, @displayName,
            @startDate, @endDate, @dailyStartTime, @dailyEndTime, @activeDaysOfWk,
            @dollarLimitPerOrder, @verificationMethod, @paymentDetails, @billingIntervalDays)
    RETURNING id`,
		pgx.NamedArgs{
			"shopId":              data.ShopId,
			"ownerId":             data.OwnerId,
			"paymentMethod":       data.PaymentMethod,
			"organization":        data.Organization,
			"displayName":         data.DisplayName,
			"startDate":           data.StartDate,
			"endDate":             data.EndDate,
			"dailyStartTime":      data.DailyStartTime,
			"dailyEndTime":        data.DailyEndTime,
			"activeDaysOfWk":      data.ActiveDaysOfWk,
			"dollarLimitPerOrder": data.DollarLimitPerOrder,
			"verificationMethod":  data.VerificationMethod,
			"paymentDetails":      data.PaymentDetails,
			"billingIntervalDays": data.BillingIntervalDays,
		})

	var tabId int
	err = row.Scan(&tabId)
	if err != nil {
		return handlePgxError(err)
	}

	err = s.setTabUsers(ctx, tx, data.ShopId, tabId, data.VerificationList)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return handlePgxError(err)
	}

	return nil
}

func (s *PgxStore) UpdateTab(ctx context.Context, shopId int, tabId int, data *types.TabUpdate) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return handlePgxError(err)
	}

	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
    UPDATE tabs SET
      (payment_method, organization, display_name,
      start_date, end_date, daily_start_time, daily_end_time, active_days_of_wk,
      dollar_limit_per_order, verification_method, payment_details, billing_interval_days) 
    = (@paymentMethod, @organization, @displayName,
            @startDate, @endDate, @dailyStartTime, @dailyEndTime, @activeDaysOfWk,
            @dollarLimitPerOrder, @verificationMethod, @paymentDetails, @billingIntervalDays)
    WHERE id = @tabId AND shop_id = @shopId`,
		pgx.NamedArgs{
			"shopId":              shopId,
			"tabId":               tabId,
			"paymentMethod":       data.PaymentMethod,
			"organization":        data.Organization,
			"displayName":         data.DisplayName,
			"startDate":           data.StartDate,
			"endDate":             data.EndDate,
			"dailyStartTime":      data.DailyStartTime,
			"dailyEndTime":        data.DailyEndTime,
			"activeDaysOfWk":      data.ActiveDaysOfWk,
			"dollarLimitPerOrder": data.DollarLimitPerOrder,
			"verificationMethod":  data.VerificationMethod,
			"paymentDetails":      data.PaymentDetails,
			"billingIntervalDays": data.BillingIntervalDays,
		})
	if err != nil {
		return handlePgxError(err)
	}

	err = s.setTabUsers(ctx, tx, shopId, tabId, data.VerificationList)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return handlePgxError(err)
	}

	return nil

}

func (s *PgxStore) ApproveTab(ctx context.Context, shopId int, tabId int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return handlePgxError(err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
    UPDATE tabs SET
      payment_method = u.payment_method,
      organization = u.organization,
      display_name = u.display_name,
      start_date = u.start_date,
      end_date = u.end_date,
      daily_start_time = u.daily_start_time,
      daily_end_time = u.daily_end_time,
      active_days_of_wk = u.active_days_of_wk,
      dollar_limit_per_order = u.dollar_limit_per_order,
      verification_method = u.verification_method,
      payment_details = u.payment_details,
      billing_interval_days = u.billing_interval_days
    FROM tab_updates AS u
    WHERE tabs.id = @tabId AND tabs.shop_id = @shopId 
      AND u.shop_id = tabs.shop_id AND u.tab_id = tabs.id`,
		pgx.NamedArgs{
			"shopId": shopId,
			"tabId":  tabId,
		})
	if err != nil {
		return handlePgxError(err)
	}
	result, err := tx.Exec(ctx, `
    UPDATE tabs SET
      status = @status
    WHERE tabs.id = @tabId AND tabs.shop_id = @shopId 
    `, pgx.NamedArgs{
		"shopId": shopId,
		"tabId":  tabId,
		"status": types.TAB_STATUS_CONFIRMED,
	})
	if err != nil {
		return handlePgxError(err)
	}

	if result.RowsAffected() == 0 {
		return services.NewNotFoundServiceError(nil)
	}

	_, err = tx.Exec(ctx, `
    DELETE FROM tab_updates
    WHERE shop_id = @shopId AND tab_id = @tabId`,
		pgx.NamedArgs{
			"shopId": shopId,
			"tabId":  tabId,
		})
	if err != nil {
		return handlePgxError(err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return handlePgxError(err)
	}
	return nil

}

func (s *PgxStore) SetTabUpdates(ctx context.Context, shopId int, tabId int, data *types.TabUpdate) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return handlePgxError(err)
	}

	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
    INSERT INTO tab_updates 
      (shop_id, tab_id, payment_method, organization, display_name,
      start_date, end_date, daily_start_time, daily_end_time, active_days_of_wk,
      dollar_limit_per_order, verification_method, payment_details, billing_interval_days) 
    VALUES (@shopId, @tabId, @paymentMethod, @organization, @displayName,
            @startDate, @endDate, @dailyStartTime, @dailyEndTime, @activeDaysOfWk,
            @dollarLimitPerOrder, @verificationMethod, @paymentDetails, @billingIntervalDays)
    ON CONFLICT (shop_id, tab_id) DO UPDATE SET
      (payment_method, organization, display_name,
      start_date, end_date, daily_start_time, daily_end_time, active_days_of_wk,
      dollar_limit_per_order, verification_method, payment_details, billing_interval_days) 
    = (excluded.payment_method, excluded.organization, excluded.display_name,
      excluded.start_date, excluded.end_date, excluded.daily_start_time, excluded.daily_end_time, excluded.active_days_of_wk,
      excluded.dollar_limit_per_order, excluded.verification_method, excluded.payment_details, excluded.billing_interval_days)`,
		pgx.NamedArgs{
			"shopId":              shopId,
			"tabId":               tabId,
			"paymentMethod":       data.PaymentMethod,
			"organization":        data.Organization,
			"displayName":         data.DisplayName,
			"startDate":           data.StartDate,
			"endDate":             data.EndDate,
			"dailyStartTime":      data.DailyStartTime,
			"dailyEndTime":        data.DailyEndTime,
			"activeDaysOfWk":      data.ActiveDaysOfWk,
			"dollarLimitPerOrder": data.DollarLimitPerOrder,
			"verificationMethod":  data.VerificationMethod,
			"paymentDetails":      data.PaymentDetails,
			"billingIntervalDays": data.BillingIntervalDays,
		})
	if err != nil {
		return handlePgxError(err)
	}

	err = s.setTabUsers(ctx, tx, shopId, tabId, data.VerificationList)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return handlePgxError(err)
	}

	return nil

}

func (s *PgxStore) GetTabs(ctx context.Context, shopId int) ([]types.Tab, error) {
	rows, err := s.pool.Query(ctx, `
    SELECT tabs.*, to_jsonb(u) - 'shop_id' - 'tab_id' as pending_updates, array_remove(array_agg(tab_users.email), null) as verification_list
    FROM tabs
    LEFT JOIN tab_updates AS u ON tabs.shop_id = u.shop_id AND tabs.id = u.tab_id
    LEFT JOIN tab_users ON tabs.shop_id = tab_users.shop_id AND tabs.id = tab_users.tab_id
    WHERE tabs.shop_id = @shopId
    GROUP BY tabs.shop_id, tabs.id, u.*`,
		pgx.NamedArgs{
			"shopId": shopId,
		})

	if err != nil {
		return nil, handlePgxError(err)
	}

	tabs, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[types.Tab])
	if err != nil {
		return nil, handlePgxError(err)
	}
	return tabs, nil
}

func (s *PgxStore) GetTab(ctx context.Context, shopId int, tabId int) (types.Tab, error) {
	rows, err := s.pool.Query(ctx, `
    SELECT tabs.*, to_jsonb(u) - 'shop_id' - 'tab_id' as pending_updates, array_remove(array_agg(tab_users.email), null) as verification_list
    FROM tabs
    LEFT JOIN tab_updates AS u ON tabs.shop_id = u.shop_id AND tabs.id = u.tab_id
    LEFT JOIN tab_users ON tabs.shop_id = tab_users.shop_id AND tabs.id = tab_users.tab_id
    WHERE tabs.shop_id = @shopId AND tabs.id = @tabId
    GROUP BY tabs.shop_id, tabs.id, u.*`,
		pgx.NamedArgs{
			"shopId": shopId,
			"tabId":  tabId,
		})

	if err != nil {
		return types.Tab{}, handlePgxError(err)
	}

	tab, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByNameLax[types.Tab])
	if err != nil {
		return types.Tab{}, handlePgxError(err)
	}
	return tab, nil
}

func (s *PgxStore) SetTabUsers(ctx context.Context, shopId int, tabId int, emails []string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return handlePgxError(err)
	}

	defer tx.Rollback(ctx)
	err = s.setTabUsers(ctx, tx, shopId, tabId, emails)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return handlePgxError(err)
	}

	return nil
}

func (s *PgxStore) setTabUsers(ctx context.Context, tx pgx.Tx, shopId int, tabId int, emails []string) error {
	_, err := tx.Exec(ctx, `
    CREATE TEMPORARY TABLE _temp_upsert_tab_users (LIKE tab_users INCLUDING ALL ) ON COMMIT DROP`)
	if err != nil {
		return handlePgxError(err)
	}

	_, err = tx.CopyFrom(ctx, pgx.Identifier{"_temp_upsert_tab_users"}, []string{"shop_id", "tab_id", "email"}, pgx.CopyFromSlice(len(emails), func(i int) ([]any, error) {
		return []any{shopId, tabId, emails[i]}, nil
	}))
	if err != nil {
		return handlePgxError(err)
	}

	_, err = tx.Exec(ctx, `
    INSERT INTO tab_users SELECT * FROM _temp_upsert_tab_users ON CONFLICT DO NOTHING`)
	if err != nil {
		return handlePgxError(err)
	}

	_, err = tx.Exec(ctx,
		`DELETE FROM tab_users WHERE shop_id = @shopId AND tab_id = @tabId AND NOT (email = ANY (@emails))`,
		pgx.NamedArgs{
			"shopId": shopId,
			"tabId":  tabId,
			"emails": emails,
		})
	if err != nil {
		return handlePgxError(err)
	}

	return nil
}

func (s *PgxStore) getTargetBill(ctx context.Context, tx pgx.Tx, shopId int, tabId int) (int, error) {
	row := tx.QueryRow(ctx, `
    SELECT b.id
    FROM tab_bills b
    LEFT JOIN tabs ON b.shop_id = tabs.shop_id AND b.tab_id = tabs.id
    WHERE b.shop_id = @shopId AND b.tab_id = @tabId
    AND b.is_paid = false AND b.start_time <= NOW() AND b.start_time + INTERVAL '1' day * tabs.billing_interval_days >= NOW() 
    ORDER BY start_time DESC
    `,
		pgx.NamedArgs{
			"shopId": shopId,
			"tabId":  tabId,
		})
	var billId int
	err := row.Scan(&billId)
	if err == nil {
		return billId, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, handlePgxError(err)
	}

	row = tx.QueryRow(ctx, `
    INSERT INTO tab_bills 
    (shop_id, tab_id, start_time) VALUES (@shopId, @tabId, NOW()) RETURNING id
    `, pgx.NamedArgs{
		"shopId": shopId,
		"tabId":  tabId,
	})
	err = row.Scan(&billId)
	if err != nil {
		return 0, handlePgxError(err)
	}

	return billId, nil
}

func (s *PgxStore) AddOrderToTab(ctx context.Context, shopId int, tabId int, data *types.OrderCreate) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return handlePgxError(err)
	}

	defer tx.Rollback(ctx)

	billId, err := s.getTargetBill(ctx, tx, shopId, tabId)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
    CREATE TEMPORARY TABLE _temp_upsert_order_items (LIKE order_items INCLUDING ALL ) ON COMMIT DROP`)
	if err != nil {
		return handlePgxError(err)
	}
	_, err = tx.Exec(ctx, `
	   CREATE TEMPORARY TABLE _temp_upsert_order_variants (LIKE order_variants INCLUDING ALL ) ON COMMIT DROP`)
	if err != nil {
		return handlePgxError(err)
	}

	type itemOrder struct {
		id       int
		quantity int
	}

	type variantOrder struct {
		itemOrder
		variantId int
	}

	itemOrders := make([]itemOrder, 0)
	variantOrders := make([]variantOrder, 0)
	for k, v := range data.Items {
		itemOrders = append(itemOrders, itemOrder{id: k, quantity: v.Quantity})
		for _, v := range v.Variants {
			variantOrders = append(variantOrders, variantOrder{itemOrder: itemOrder{id: k, quantity: v.Quantity}, variantId: v.Id})
		}
	}

	_, err = tx.CopyFrom(ctx, pgx.Identifier{"_temp_upsert_order_items"},
		[]string{"shop_id", "tab_id", "bill_id", "item_id", "quantity"}, pgx.CopyFromSlice(len(itemOrders), func(i int) ([]any, error) {
			return []any{shopId, tabId, billId, itemOrders[i].id, itemOrders[i].quantity}, nil
		}))
	if err != nil {
		return handlePgxError(err)
	}

	_, err = tx.CopyFrom(ctx, pgx.Identifier{"_temp_upsert_order_variants"},
		[]string{"shop_id", "tab_id", "bill_id", "item_id", "variant_id", "quantity"}, pgx.CopyFromSlice(len(variantOrders), func(i int) ([]any, error) {
			return []any{shopId, tabId, billId, variantOrders[i].id, variantOrders[i].variantId, variantOrders[i].quantity}, nil
		}))
	if err != nil {
		return handlePgxError(err)
	}

	_, err = tx.Exec(ctx, `
    INSERT INTO order_items SELECT * FROM _temp_upsert_order_items ON CONFLICT (shop_id, tab_id, bill_id, item_id) DO UPDATE
    SET quantity = order_items.quantity + excluded.quantity`)
	if err != nil {
		return handlePgxError(err)
	}
	_, err = tx.Exec(ctx, `
	   INSERT INTO order_variants SELECT * FROM _temp_upsert_order_variants ON CONFLICT (shop_id, tab_id, bill_id, item_id, variant_id) DO UPDATE
	   SET quantity = order_variants.quantity + excluded.quantity`)
	if err != nil {
		return handlePgxError(err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return handlePgxError(err)
	}

	return nil
}

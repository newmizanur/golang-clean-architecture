package resolver

import (
	"context"
	"fmt"
	"strconv"

	"golang-clean-architecture/internal/delivery/graphql/currencyservice"
	"golang-clean-architecture/internal/delivery/graphql/dataloader"
	"golang-clean-architecture/internal/delivery/graphql/graph"
	"golang-clean-architecture/internal/delivery/graphql/graph/model"
	pb "golang-clean-architecture/internal/delivery/grpc/pb"

	"google.golang.org/grpc/metadata"
)

// grpcCtx appends the JWT from the GraphQL HTTP request to outgoing gRPC metadata.
func grpcCtx(ctx context.Context) context.Context {
	token := jwtFromContext(ctx)
	if token == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
}

// Query resolvers

func (r *queryResolver) Item(ctx context.Context, id string) (*model.Item, error) {
	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %s", id)
	}
	resp, err := r.ItemClient.GetItem(grpcCtx(ctx), &pb.GetItemRequest{Id: idInt})
	if err != nil {
		return nil, err
	}
	return pbToModel(resp), nil
}

func (r *queryResolver) Items(ctx context.Context, filter model.ListItemsInput) (*model.ItemPage, error) {
	req := &pb.ListItemsRequest{
		Page: int32(filter.Page),
		Size: int32(filter.Size),
	}
	if filter.Name != nil {
		req.Name = *filter.Name
	}
	if filter.Sku != nil {
		req.Sku = *filter.Sku
	}
	if filter.Sort != nil {
		req.Sort = *filter.Sort
	}

	resp, err := r.ItemClient.ListItems(grpcCtx(ctx), req)
	if err != nil {
		return nil, err
	}

	items := make([]*model.Item, len(resp.Items))
	for i, it := range resp.Items {
		items[i] = pbToModel(it)
	}
	return &model.ItemPage{Items: items, Total: int(resp.Total)}, nil
}

// Mutation resolvers

func (r *mutationResolver) CreateItem(ctx context.Context, input model.CreateItemInput) (*model.Item, error) {
	resp, err := r.ItemClient.CreateItem(grpcCtx(ctx), &pb.CreateItemRequest{
		Name:     input.Name,
		Sku:      input.Sku,
		Currency: input.Currency,
		Stock:    int32(input.Stock),
	})
	if err != nil {
		return nil, err
	}
	return pbToModel(resp), nil
}

// Item field resolver — Currency (NAIVE: one call per item = N+1)

func (r *itemResolver) Currency(ctx context.Context, obj *model.Item) (*model.Currency, error) {
	loader := dataloader.LoaderFrom(ctx)
	if loader == nil {
		// Fallback for contexts without middleware (e.g. tests).
		c, err := currencyservice.GetCurrency(ctx, obj.CurrencyCode)
		if err != nil {
			return nil, err
		}
		return &model.Currency{Code: c.Code, Symbol: c.Symbol, DecimalPlaces: c.DecimalPlaces}, nil
	}
	c, err := loader.Load(ctx, obj.CurrencyCode)
	if err != nil {
		return nil, err
	}
	return &model.Currency{Code: c.Code, Symbol: c.Symbol, DecimalPlaces: c.DecimalPlaces}, nil
}

// Helper

func pbToModel(r *pb.ItemResponse) *model.Item {
	return &model.Item{
		ID:           fmt.Sprintf("%d", r.Id),
		Name:         r.Name,
		Sku:          r.Sku,
		Stock:        int(r.Stock),
		CurrencyCode: r.Currency,
	}
}

// Boilerplate: implement graph.ResolverRoot

func (r *Resolver) Query() graph.QueryResolver       { return &queryResolver{r} }
func (r *Resolver) Mutation() graph.MutationResolver { return &mutationResolver{r} }
func (r *Resolver) Item() graph.ItemResolver         { return &itemResolver{r} }

type queryResolver    struct{ *Resolver }
type mutationResolver struct{ *Resolver }
type itemResolver     struct{ *Resolver }

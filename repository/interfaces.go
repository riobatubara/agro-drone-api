package repository

import "context"

//go:generate mockgen -source=interfaces.go -destination=interfaces.mock.gen.go -package=repository
type RepositoryInterface interface {
	GetTestById(ctx context.Context, input GetTestByIdInput) (output GetTestByIdOutput, err error)
	CreateEstate(ctx context.Context, input CreateEstateInput) (output CreateEstateOutput, err error)
	GetEstate(ctx context.Context) (output GetEstateOutput, err error)
	GetEstateById(ctx context.Context, input GetEstateByIdInput) (output GetEstateByIdOutput, err error)
	CreateTree(ctx context.Context, input CreateTreeInput) (err error)
	GetTreeHeightsById(ctx context.Context, input GetTreeHeightsByIdInput) (output GetTreeHeightsByIdOutput, err error)
	GetTreeMapById(ctx context.Context, input GetTreeMapByIdInput) (output GetTreeMapByIdOutput, err error)
}

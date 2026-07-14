package repository

import (
	"context"
	"strconv"
)

func (r *Repository) GetTestById(ctx context.Context, input GetTestByIdInput) (output GetTestByIdOutput, err error) {
	err = r.Db.QueryRowContext(ctx, "SELECT name FROM test WHERE id = $1", input.Id).Scan(&output.Name)
	return
}

func (r *Repository) CreateEstate(ctx context.Context, input CreateEstateInput) (output CreateEstateOutput, err error) {
	query := `INSERT INTO estates (width, length) VALUES ($1, $2) RETURNING id`
	err = r.Db.QueryRowContext(ctx, query, input.Width, input.Length).Scan(&output.Id)
	return
}

func (r *Repository) GetEstate(ctx context.Context) (output GetEstateOutput, err error) {
	query := `SELECT id FROM estates`
	rows, err := r.Db.QueryContext(ctx, query)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return
		}
		output.Ids = append(output.Ids, id)
	}
	err = rows.Err()
	return
}

func (r *Repository) GetEstateById(ctx context.Context, input GetEstateByIdInput) (output GetEstateByIdOutput, err error) {
	query := `SELECT width, length FROM estates WHERE id = $1`
	err = r.Db.QueryRowContext(ctx, query, input.Id).Scan(&output.Width, &output.Length)
	return
}

func (r *Repository) CreateTree(ctx context.Context, input CreateTreeInput) (err error) {
	var id string
	query := `INSERT INTO trees (estate_id, x, y, height) VALUES ($1, $2, $3, $4) RETURNING id`
	err = r.Db.QueryRowContext(ctx, query, input.EstateID, input.X, input.Y, input.Height).Scan(&id)
	return
}

func (r *Repository) GetTreeHeightsById(ctx context.Context, input GetTreeHeightsByIdInput) (output GetTreeHeightsByIdOutput, err error) {
	query := `SELECT height FROM trees WHERE estate_id = $1`
	rows, err := r.Db.QueryContext(ctx, query, input.EstateID)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var h int
		if err = rows.Scan(&h); err != nil {
			return
		}
		output.Height = append(output.Height, h)
	}
	err = rows.Err()
	return
}

func (r *Repository) GetTreeMapById(ctx context.Context, input GetTreeMapByIdInput) (output GetTreeMapByIdOutput, err error) {
	query := `SELECT x, y, height FROM trees WHERE estate_id = $1`
	rows, err := r.Db.QueryContext(ctx, query, input.EstateID)
	if err != nil {
		return
	}
	defer rows.Close()

	output.Key = make(map[string]int)
	for rows.Next() {
		var tx, ty, th int
		if err = rows.Scan(&tx, &ty, &th); err != nil {
			return
		}
		key := strconv.Itoa(tx) + "," + strconv.Itoa(ty)
		output.Key[key] = th
	}
	err = rows.Err()
	return
}

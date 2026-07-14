package repository

type GetTestByIdInput struct{ Id string }
type GetTestByIdOutput struct{ Name string }

type CreateEstateInput struct{ Width, Length int }
type GetEstateOutput struct{ Ids []string } // Fixed spelling typo

type GetEstateByIdInput struct{ Id string }
type GetEstateByIdOutput struct{ Width, Length int }

type CreateTreeInput struct {
	EstateID string
	X        int
	Y        int
	Height   int
}

type GetTreeHeightsByIdInput struct{ EstateID string }
type GetTreeHeightsByIdOutput struct{ Height []int }

type GetTreeMapByIdInput struct{ EstateID string }
type GetTreeMapByIdOutput struct{ Key map[string]int }

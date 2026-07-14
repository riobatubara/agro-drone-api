package handler

import (
	"net/http"
	"sort"
	"strconv"

	"agro-drone-api/generated"
	"agro-drone-api/repository"

	"github.com/labstack/echo/v4"
)

// POST /estate
func (s *Server) CreateEstate(c echo.Context) error {
	var input repository.CreateEstateInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid payload schema"})
	}

	if input.Width <= 0 || input.Length <= 0 {
		return c.JSON(http.StatusUnprocessableEntity, map[string]string{"error": "dimensions must be greater than zero"})
	}

	output, err := s.Repository.CreateEstate(c.Request().Context(), input)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]string{"id": output.Id})
}

// POST /estate/:id/tree
func (s *Server) CreateTree(c echo.Context, id string) error {
	estateID := id

	var input repository.CreateTreeInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid payload schema"})
	}
	input.EstateID = estateID

	// Core data boundaries structural safety validation
	estateData, err := s.Repository.GetEstateById(c.Request().Context(), repository.GetEstateByIdInput{Id: estateID})
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "target estate structure missing from grid registry"})
	}

	if input.X < 1 || input.X > estateData.Width || input.Y < 1 || input.Y > estateData.Length {
		return c.JSON(http.StatusUnprocessableEntity, map[string]string{"error": "coordinates fall outside estate boundaries"})
	}

	// Validate Tree Height (Must range between 1 and 30 meters inclusive)
	if input.Height < 1 || input.Height > 30 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tree height must be between 1 and 30 meters inclusive"})
	}

	// Fetch existing tree layout to check for spatial coordinate overlaps
	treeMapData, err := s.Repository.GetTreeMapById(c.Request().Context(), repository.GetTreeMapByIdInput{EstateID: estateID})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Format coordinate match key to fit "x,y" pattern template
	coordinateKey := strconv.Itoa(input.X) + "," + strconv.Itoa(input.Y)
	if _, duplicateExists := treeMapData.Key[coordinateKey]; duplicateExists {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "plot already has tree"})
	}

	// Save clean record into storage layer
	if err := s.Repository.CreateTree(c.Request().Context(), input); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.NoContent(http.StatusCreated)
}

// GET /estate/:id/stats
func (s *Server) GetEstateStats(c echo.Context, id string) error {
	estateID := id

	// Verify estate exists
	_, err := s.Repository.GetEstateById(c.Request().Context(), repository.GetEstateByIdInput{Id: estateID})
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "estate not found"})
	}

	// Fetch all tree heights for the estate
	heightsData, err := s.Repository.GetTreeHeightsById(c.Request().Context(), repository.GetTreeHeightsByIdInput{EstateID: estateID})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Handle empty datasets safely to prevent division by zero or panic
	if len(heightsData.Height) == 0 {
		return c.JSON(http.StatusOK, map[string]any{
			"count":  0,
			"max":    0,
			"min":    0,
			"median": 0,
		})
	}

	// Sort heights in ascending order for min, max, and median calculations
	sort.Ints(heightsData.Height)

	totalTrees := len(heightsData.Height)
	min := heightsData.Height[0]
	max := heightsData.Height[totalTrees-1]

	// Calculate Median
	var median float64
	mid := totalTrees / 2

	if totalTrees%2 != 0 {
		// Odd dataset: choose the single center value
		median = float64(heightsData.Height[mid])
	} else {
		// Even dataset: average the two values flanking the center point
		median = float64(heightsData.Height[mid-1]+heightsData.Height[mid]) / 2.0
	}

	return c.JSON(http.StatusOK, map[string]any{
		"count":  totalTrees,
		"max":    max,
		"min":    min,
		"median": median,
	})
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// GET /estate/:id/drone-plan
func (s *Server) GetDronePlan(c echo.Context, id string, params generated.GetDronePlanParams) error {
	estateID := id

	// Fetch Estate Dimensions
	estateData, err := s.Repository.GetEstateById(c.Request().Context(), repository.GetEstateByIdInput{Id: estateID})
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "estate not found"})
	}

	// Fetch Tree Maps for heights
	treeMapData, err := s.Repository.GetTreeMapById(c.Request().Context(), repository.GetTreeMapByIdInput{EstateID: estateID})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	//  Parse Optional Max Distance
	var maxDistance int
	hasMaxDistance := params.MaxDistance != nil
	if hasMaxDistance {
		maxDistance = *params.MaxDistance
	}

	// Generate ordered sequence of coordinates following the required snake path
	type Point struct{ X, Y int }
	var path []Point

	for y := 1; y <= estateData.Length; y++ {
		if y%2 != 0 {
			// Odd rows move West to East
			for x := 1; x <= estateData.Width; x++ {
				path = append(path, Point{X: x, Y: y})
			}
		} else {
			// Even rows move East to West
			for x := estateData.Width; x >= 1; x-- {
				path = append(path, Point{X: x, Y: y})
			}
		}
	}

	totalDistance := 0
	currentZ := 0 // Drone starts on ground level (0 meters)
	var landPoint Point
	forcedLanding := false

	// Helper function to fetch target hovering altitude (Tree height + 1, or 1 if empty)
	getTargetZ := func(pt Point) int {
		key := strconv.Itoa(pt.X) + "," + strconv.Itoa(pt.Y)
		if h, exists := treeMapData.Key[key]; exists {
			return h + 1
		}
		return 1
	}

	for i, pt := range path {
		targetZ := getTargetZ(pt)

		// Vertical movement to align with the current plot target height
		vDist := abs(targetZ - currentZ)
		if hasMaxDistance && totalDistance+vDist > maxDistance {
			landPoint = pt
			forcedLanding = true
			break
		}
		totalDistance += vDist
		currentZ = targetZ

		// Horizontal movement to the next plot (if there is a next plot)
		if i < len(path)-1 {
			nextPt := path[i+1]
			// Horizontal distance is always 10 meters per plot increment
			hDist := (abs(nextPt.X-pt.X) + abs(nextPt.Y-pt.Y)) * 10

			if hasMaxDistance && totalDistance+hDist > maxDistance {
				landPoint = pt // Forced landing happens on the current plot before moving
				forcedLanding = true
				break
			}
			totalDistance += hDist
		}
	}

	// Descent to ground level after monitoring the final plot (if not forced down early)
	if !forcedLanding && len(path) > 0 {
		finalVDist := currentZ // distance back to 0m
		if hasMaxDistance && totalDistance+finalVDist > maxDistance {
			landPoint = path[len(path)-1]
			forcedLanding = true
		} else {
			totalDistance += finalVDist
			landPoint = path[len(path)-1] // completes path, lands at the final point
		}
	}

	// Structure response payload to match assignment criteria
	if hasMaxDistance {
		return c.JSON(http.StatusOK, map[string]any{
			"distance": totalDistance,
			"rest": map[string]int{
				"x": landPoint.X,
				"y": landPoint.Y,
			},
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"distance": totalDistance,
	})
}

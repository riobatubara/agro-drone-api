package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"agro-drone-api/generated"
	"agro-drone-api/handler"
	"agro-drone-api/repository"

	"github.com/labstack/echo/v4"
)

// setupTestEnvironment initializes a real transient repository instance
// linked against the test database container defined in your docker-compose environment.
func setupTestEnvironment(t *testing.T) (*echo.Echo, repository.RepositoryInterface) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Fallback local connection fallback configuration parameters
		dsn = "postgres://postgres:postgres@localhost:5432/database?sslmode=disable"
	}

	repoOpts := repository.NewRepositoryOptions{Dsn: dsn}
	repo := repository.NewRepository(repoOpts)

	// Clean out prior state tracking to guarantee test idempotency bounds
	ctx := context.Background()
	_, _ = repo.Db.ExecContext(ctx, "TRUNCATE TABLE trees CASCADE;")
	_, _ = repo.Db.ExecContext(ctx, "TRUNCATE TABLE estates CASCADE;")

	e := echo.New()
	s := handler.NewServer(handler.NewServerOptions{Repository: repo})

	e.POST("/estate", s.CreateEstate)

	e.POST("/estate/:id/tree", func(c echo.Context) error {
		return s.CreateTree(c, c.Param("id"))
	})

	e.GET("/estate/:id/stats", func(c echo.Context) error {
		return s.GetEstateStats(c, c.Param("id"))
	})

	e.GET("/estate/:id/drone-plan", func(c echo.Context) error {
		var maxDistPtr *int
		if q := c.QueryParam("max_distance"); q != "" {
			if val, err := strconv.Atoi(q); err == nil {
				maxDistPtr = &val
			}
		}

		params := generated.GetDronePlanParams{
			MaxDistance: maxDistPtr,
		}
		return s.GetDronePlan(c, c.Param("id"), params)
	})

	return e, repo
}

func TestFullAgroDronePipelineIntegration(t *testing.T) {
	e, _ := setupTestEnvironment(t)

	estatePayload := `{"width": 5, "length": 1}`
	req := httptest.NewRequest(http.MethodPost, "/estate", bytes.NewBufferString(estatePayload))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Step 1 Failed: Expected status 201, got %d. Response: %s", rec.Code, rec.Body.String())
	}

	var estateRes map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &estateRes); err != nil {
		t.Fatalf("Failed to parse estate generation response: %v", err)
	}

	estateID := estateRes["id"]
	if estateID == "" {
		t.Fatal("Step 1 Failed: Received empty estate identification footprint UUID key token")
	}

	treesToPlant := []string{
		`{"x": 2, "y": 1, "height": 5}`, // Z Target -> 6 meters
		`{"x": 3, "y": 1, "height": 3}`, // Z Target -> 4 meters
		`{"x": 4, "y": 1, "height": 4}`, // Z Target -> 5 meters
	}

	for _, payload := range treesToPlant {
		req = httptest.NewRequest(http.MethodPost, "/estate/"+estateID+"/tree", bytes.NewBufferString(payload))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec = httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("Step 2 Failed: Plant request failed for %s with code %d", payload, rec.Code)
		}
	}

	duplicatePayload := `{"x": 2, "y": 1, "height": 10}`
	req = httptest.NewRequest(http.MethodPost, "/estate/"+estateID+"/tree", bytes.NewBufferString(duplicatePayload))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Step 3 Failed: Expected duplicate validation error status 400, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/estate/"+estateID+"/stats", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Step 4 Failed: Expected stats status 200, got %d", rec.Code)
	}

	var statsRes map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &statsRes); err != nil {
		t.Fatalf("Failed to parse statistics payload: %v", err)
	}

	// Heights: [3, 4, 5] -> Middle single element should yield median 4.0
	if statsRes["count"].(float64) != 3 || statsRes["min"].(float64) != 3 || statsRes["max"].(float64) != 5 || statsRes["median"].(float64) != 4 {
		t.Errorf("Step 4 Failed: Incorrect analytics payload layout values returned: %v", statsRes)
	}

	req = httptest.NewRequest(http.MethodGet, "/estate/"+estateID+"/drone-plan", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Step 5 Failed: Expected drone plan status 200, got %d", rec.Code)
	}

	var planRes map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &planRes); err != nil {
		t.Fatalf("Failed to unmarshal drone evaluation: %v", err)
	}

	// Must measure exactly 54 meters according to assignment parameters definition criteria
	if planRes["distance"].(float64) != 54 {
		t.Errorf("Step 5 Failed: Expected exact tracking flight length distance matrix to scale 54, got %v", planRes["distance"])
	}

	req = httptest.NewRequest(http.MethodGet, "/estate/"+estateID+"/drone-plan?max_distance=20", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Step 6 Failed: Expected status 200, got %d", rec.Code)
	}

	var bonusRes map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &bonusRes); err != nil {
		t.Fatalf("Failed to unmarshal bonus capacity map payload context: %v", err)
	}

	// Forced landing must contain coordinates mapping bounds
	if bonusRes["rest"] == nil {
		t.Errorf("Step 6 Failed: Expected 'rest' coordinates response object payload but got nil")
	}
}

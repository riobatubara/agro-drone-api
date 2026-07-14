package handler_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"agro-drone-api/handler"
	"agro-drone-api/repository"

	"github.com/labstack/echo/v4"
	"go.uber.org/mock/gomock"
)

func TestCreateEstate(t *testing.T) {
	tests := []struct {
		name          string
		payload       string
		setupMock     func(m *repository.MockRepositoryInterface)
		expectedCode  int
		expectedInRes string
	}{
		{
			name:    "Success Case",
			payload: `{"width": 10, "length": 20}`,
			setupMock: func(m *repository.MockRepositoryInterface) {
				m.EXPECT().
					CreateEstate(gomock.Any(), repository.CreateEstateInput{Width: 10, Length: 20}).
					Return(repository.CreateEstateOutput{Id: "generated-uuid-123"}, nil).
					Times(1)
			},
			expectedCode:  http.StatusCreated,
			expectedInRes: `{"id":"generated-uuid-123"}`,
		},
		{
			name:         "Invalid JSON Payload",
			payload:      `{"width": "invalid", "length": 20}`,
			setupMock:    func(m *repository.MockRepositoryInterface) {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Dimensions Zero or Less Rejected",
			payload:      `{"width": 0, "length": 20}`,
			setupMock:    func(m *repository.MockRepositoryInterface) {},
			expectedCode: http.StatusUnprocessableEntity,
		},
		{
			name:    "Database Layer Internal Error",
			payload: `{"width": 10, "length": 20}`,
			setupMock: func(m *repository.MockRepositoryInterface) {
				m.EXPECT().
					CreateEstate(gomock.Any(), gomock.Any()).
					Return(repository.CreateEstateOutput{}, errors.New("db disconnect")).
					Times(1)
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := repository.NewMockRepositoryInterface(ctrl)
			tt.setupMock(mockRepo)

			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/estate", bytes.NewBufferString(tt.payload))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			s := handler.NewServer(handler.NewServerOptions{Repository: mockRepo})

			if err := s.CreateEstate(c); err != nil {
				t.Fatalf("handler errored: %v", err)
			}

			if rec.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, rec.Code)
			}
			if tt.expectedInRes != "" {
				actualBody := strings.TrimSpace(rec.Body.String())
				if actualBody != tt.expectedInRes {
					t.Errorf("expected response %s, got %s", tt.expectedInRes, actualBody)
				}
			}
		})
	}
}

func TestCreateTree(t *testing.T) {
	tests := []struct {
		name          string
		estateID      string
		payload       string
		setupMock     func(m *repository.MockRepositoryInterface)
		expectedCode  int
		expectedInRes string
	}{
		{
			name:     "Success Within Boundaries",
			estateID: "estate-1",
			payload:  `{"x": 2, "y": 3, "height": 5}`,
			setupMock: func(m *repository.MockRepositoryInterface) {
				m.EXPECT().
					GetEstateById(gomock.Any(), repository.GetEstateByIdInput{Id: "estate-1"}).
					Return(repository.GetEstateByIdOutput{Width: 5, Length: 5}, nil).
					Times(1)
				m.EXPECT().
					CreateTree(gomock.Any(), repository.CreateTreeInput{EstateID: "estate-1", X: 2, Y: 3, Height: 5}).
					Return(nil).
					Times(1)
			},
			expectedCode: http.StatusCreated,
		},
		{
			name:     "Out of Bounds Error",
			estateID: "estate-1",
			payload:  `{"x": 10, "y": 3, "height": 5}`,
			setupMock: func(m *repository.MockRepositoryInterface) {
				m.EXPECT().
					GetEstateById(gomock.Any(), gomock.Any()).
					Return(repository.GetEstateByIdOutput{Width: 5, Length: 5}, nil).
					Times(1)
			},
			expectedCode:  http.StatusUnprocessableEntity,
			expectedInRes: "coordinates fall outside estate boundaries",
		},
		{
			name:     "Estate Missing From Registry",
			estateID: "estate-missing",
			payload:  `{"x": 2, "y": 3, "height": 5}`,
			setupMock: func(m *repository.MockRepositoryInterface) {
				m.EXPECT().
					GetEstateById(gomock.Any(), gomock.Any()).
					Return(repository.GetEstateByIdOutput{}, errors.New("not found")).
					Times(1)
			},
			expectedCode:  http.StatusNotFound,
			expectedInRes: "target estate structure missing from grid registry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := repository.NewMockRepositoryInterface(ctrl)
			tt.setupMock(mockRepo)

			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tt.payload))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/estate/:id/tree")
			c.SetParamNames("id")
			c.SetParamValues(tt.estateID)

			s := handler.NewServer(handler.NewServerOptions{Repository: mockRepo})

			if err := s.CreateTree(c); err != nil {
				t.Fatalf("handler errored unexpectedly: %v", err)
			}

			if rec.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, rec.Code)
			}
			if tt.expectedInRes != "" && !strings.Contains(rec.Body.String(), tt.expectedInRes) {
				t.Errorf("expected response to contain %q, got %q", tt.expectedInRes, rec.Body.String())
			}
		})
	}
}

func TestCreateTree_CoordinateValidation(t *testing.T) {
	tests := []struct {
		name          string
		payload       string
		setupMock     func(m *repository.MockRepositoryInterface)
		expectedCode  int
		expectedInRes string
	}{
		{
			name:    "Rejected: Duplicate Plot Coordinate Error",
			payload: `{"x": 2, "y": 3, "height": 10}`,
			setupMock: func(m *repository.MockRepositoryInterface) {
				// Mock estate data verification boundary check
				m.EXPECT().
					GetEstateById(gomock.Any(), repository.GetEstateByIdInput{Id: "estate-valid"}).
					Return(repository.GetEstateByIdOutput{Width: 10, Length: 10}, nil).
					Times(1)

				// Mock return with a tree already present at (2,3)
				mockExistingTrees := map[string]int{"2,3": 15}
				m.EXPECT().
					GetTreeMapById(gomock.Any(), repository.GetTreeMapByIdInput{EstateID: "estate-valid"}).
					Return(repository.GetTreeMapByIdOutput{Key: mockExistingTrees}, nil).
					Times(1)
			},
			expectedCode:  http.StatusBadRequest,
			expectedInRes: "plot already has tree",
		},
		{
			name:    "Accepted: Unique Coordinate Inhabiting Clean Plot Space",
			payload: `{"x": 4, "y": 4, "height": 12}`,
			setupMock: func(m *repository.MockRepositoryInterface) {
				m.EXPECT().
					GetEstateById(gomock.Any(), repository.GetEstateByIdInput{Id: "estate-valid"}).
					Return(repository.GetEstateByIdOutput{Width: 10, Length: 10}, nil).
					Times(1)

				// Mock return with other trees, but leaving (4,4) vacant
				mockExistingTrees := map[string]int{"2,3": 15}
				m.EXPECT().
					GetTreeMapById(gomock.Any(), repository.GetTreeMapByIdInput{EstateID: "estate-valid"}).
					Return(repository.GetTreeMapByIdOutput{Key: mockExistingTrees}, nil).
					Times(1)

				// Expect tree creation to execute cleanly
				m.EXPECT().
					CreateTree(gomock.Any(), repository.CreateTreeInput{EstateID: "estate-valid", X: 4, Y: 4, Height: 12}).
					Return(nil).
					Times(1)
			},
			expectedCode: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := repository.NewMockRepositoryInterface(ctrl)
			tt.setupMock(mockRepo)

			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tt.payload))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/estate/:id/tree")
			c.SetParamNames("id")
			c.SetParamValues("estate-valid")

			s := handler.NewServer(handler.NewServerOptions{Repository: mockRepo})

			if err := s.CreateTree(c); err != nil {
				t.Fatalf("handler errored unexpectedly: %v", err)
			}

			if rec.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, rec.Code)
			}
			if tt.expectedInRes != "" && !strings.Contains(rec.Body.String(), tt.expectedInRes) {
				t.Errorf("expected response to contain %q, got %q", tt.expectedInRes, rec.Body.String())
			}
		})
	}
}

func TestGetEstateStats(t *testing.T) {
	tests := []struct {
		name          string
		estateID      string
		setupMock     func(m *repository.MockRepositoryInterface)
		expectedCode  int
		expectedInRes string
	}{
		{
			name:     "Success With Valid Tree Heights",
			estateID: "estate-1",
			setupMock: func(m *repository.MockRepositoryInterface) {
				m.EXPECT().
					GetTreeHeightsById(gomock.Any(), repository.GetTreeHeightsByIdInput{EstateID: "estate-1"}).
					Return(repository.GetTreeHeightsByIdOutput{Height: []int{10, 20, 30}}, nil).
					Times(1)
			},
			expectedCode:  http.StatusOK,
			expectedInRes: `"average":20,"count":3,"max":30,"min":10`,
		},
		{
			name:     "Empty Dataset Fallback Safeties",
			estateID: "estate-empty",
			setupMock: func(m *repository.MockRepositoryInterface) {
				m.EXPECT().
					GetTreeHeightsById(gomock.Any(), gomock.Any()).
					Return(repository.GetTreeHeightsByIdOutput{Height: []int{}}, nil).
					Times(1)
			},
			expectedCode:  http.StatusOK,
			expectedInRes: `"average":0,"count":0,"max":0,"min":0`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := repository.NewMockRepositoryInterface(ctrl)
			tt.setupMock(mockRepo)

			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/estate/:id/stats")
			c.SetParamNames("id")
			c.SetParamValues(tt.estateID)

			s := handler.NewServer(handler.NewServerOptions{Repository: mockRepo})

			if err := s.GetEstateStats(c); err != nil {
				t.Fatalf("handler errored unexpectedly: %v", err)
			}

			if rec.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, rec.Code)
			}
			// Clean up string spacing variance issues during matching checks
			bodyClean := strings.ReplaceAll(rec.Body.String(), " ", "")
			if tt.expectedInRes != "" && !strings.Contains(bodyClean, tt.expectedInRes) {
				t.Errorf("expected response to contain %q, got %q", tt.expectedInRes, bodyClean)
			}
		})
	}
}

func TestGetEstateStats_MedianScenarios(t *testing.T) {
	tests := []struct {
		name          string
		mockHeights   []int
		expectedInRes string
	}{
		{
			name:          "Odd dataset - Direct Center Value",
			mockHeights:   []int{15, 5, 20}, // Sorted: 5, 15, 20
			expectedInRes: `"count":3,"max":20,"median":15,"min":5`,
		},
		{
			name:          "Even dataset - Average of Middle Elements",
			mockHeights:   []int{10, 20, 30, 40}, // Sorted: 10, 20, 30, 40 -> (20+30)/2
			expectedInRes: `"count":4,"max":40,"median":25,"min":10`,
		},
		{
			name:          "Even dataset producing decimal fractional median",
			mockHeights:   []int{4, 1, 7, 3}, // Sorted: 1, 3, 4, 7 -> (3+4)/2 = 3.5
			expectedInRes: `"count":4,"max":7,"median":3.5,"min":1`,
		},
		{
			name:          "Empty dataset fallback",
			mockHeights:   []int{},
			expectedInRes: `"count":0,"max":0,"median":0,"min":0`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := repository.NewMockRepositoryInterface(ctrl)

			// Estate verification mock
			mockRepo.EXPECT().
				GetEstateById(gomock.Any(), repository.GetEstateByIdInput{Id: "estate-stats-id"}).
				Return(repository.GetEstateByIdOutput{Width: 10, Length: 10}, nil).
				Times(1)

			// Tree height dataset mock
			mockRepo.EXPECT().
				GetTreeHeightsById(gomock.Any(), repository.GetTreeHeightsByIdInput{EstateID: "estate-stats-id"}).
				Return(repository.GetTreeHeightsByIdOutput{Height: tt.mockHeights}, nil).
				Times(1)

			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/estate/:id/stats")
			c.SetParamNames("id")
			c.SetParamValues("estate-stats-id")

			s := handler.NewServer(handler.NewServerOptions{Repository: mockRepo})

			if err := s.GetEstateStats(c); err != nil {
				t.Fatalf("handler threw an unexpected error: %v", err)
			}

			if rec.Code != http.StatusOK {
				t.Errorf("expected 200, got %d", rec.Code)
			}

			bodyClean := strings.ReplaceAll(rec.Body.String(), " ", "")
			if !strings.Contains(bodyClean, tt.expectedInRes) {
				t.Errorf("expected response breakdown to contain %q, but got %q", tt.expectedInRes, bodyClean)
			}
		})
	}
}

func TestGetDronePlan_AssignmentSample(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockRepositoryInterface(ctrl)

	// Mocking assignment background example scenario (5x1 estate)
	mockRepo.EXPECT().
		GetEstateById(gomock.Any(), repository.GetEstateByIdInput{Id: "sample-estate"}).
		Return(repository.GetEstateByIdOutput{Width: 5, Length: 1}, nil).
		Times(1)

	mockMap := map[string]int{
		"2,1": 5, // Target Z = 6m
		"3,1": 3, // Target Z = 4m
		"4,1": 4, // Target Z = 5m
	}
	mockRepo.EXPECT().
		GetTreeMapById(gomock.Any(), repository.GetTreeMapByIdInput{EstateID: "sample-estate"}).
		Return(repository.GetTreeMapByIdOutput{Key: mockMap}, nil).
		Times(1)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.SetPath("/estate/:id/drone-plan")
	ctx.SetParamNames("id")
	ctx.SetParamValues("sample-estate")

	s := handler.NewServer(handler.NewServerOptions{Repository: mockRepo})

	if err := s.GetDronePlan(ctx); err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// The assignment doc states this exact configuration must equal 54 meters!
	expectedJSON := `{"distance":54}`
	actualBody := strings.TrimSpace(rec.Body.String())
	if actualBody != expectedJSON {
		t.Errorf("expected JSON payload %s, but got %s", expectedJSON, actualBody)
	}
}

// 檔案路徑: internal/api/handlers/job_grade_list_test.go
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/erinchen11/hr-system/internal/interfaces/mocks"
	"github.com/erinchen11/hr-system/internal/models"
	common "github.com/erinchen11/hr-system/internal/models/common"
	"github.com/gin-gonic/gin"
	gomock "github.com/golang/mock/gomock"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListJobGradesHandler_ListJobGrades(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 測試資料
	testJobGrades := []models.JobGrade{
		{
			Code:        "P1",
			Name:        "Junior Engineer",
			Description: "Entry level engineer",
			MinSalary:   decimal.NewFromInt(30000),
			MaxSalary:   decimal.NewFromInt(50000),
		},
		{
			Code:        "M1",
			Name:        "Manager",
			Description: "Mid level manager",
			MinSalary:   decimal.NewFromInt(60000),
			MaxSalary:   decimal.NewFromInt(90000),
		},
	}

	hrClaims := &models.Claims{UserID: "test-hr-id", Role: models.RoleHR}
	superAdminClaims := &models.Claims{UserID: "test-super-id", Role: models.RoleSuperAdmin}
	employeeClaims := &models.Claims{UserID: "test-emp-id", Role: models.RoleEmployee}

	type testCase struct {
		name               string
		callerClaims       interface{}
		setupMocks         func(mockSvc *mocks.MockJobGradeService)
		expectedStatusCode int
		expectedAppCode    int
		expectedMessage    string
		expectData         bool
	}

	testCases := []testCase{
		{
			name:         "Success - HR can list job grades",
			callerClaims: hrClaims,
			setupMocks: func(mockSvc *mocks.MockJobGradeService) {
				mockSvc.EXPECT().ListJobGrades(gomock.Any()).Return(testJobGrades, nil)
			},
			expectedStatusCode: http.StatusOK,
			expectedAppCode:    http.StatusOK,
			expectedMessage:    "Success",
			expectData:         true,
		},
		{
			name:         "Success - SuperAdmin can list job grades",
			callerClaims: superAdminClaims,
			setupMocks: func(mockSvc *mocks.MockJobGradeService) {
				mockSvc.EXPECT().ListJobGrades(gomock.Any()).Return(testJobGrades, nil)
			},
			expectedStatusCode: http.StatusOK,
			expectedAppCode:    http.StatusOK,
			expectedMessage:    "Success",
			expectData:         true,
		},
		{
			name:               "Forbidden - Employee tries to list",
			callerClaims:       employeeClaims,
			expectedStatusCode: http.StatusForbidden,
			expectedAppCode:    http.StatusForbidden,
			expectedMessage:    "Permission denied: Only HR or Super Admin can list job grades",
		},
		{
			name:               "Unauthorized - No claims in context",
			callerClaims:       nil,
			expectedStatusCode: http.StatusUnauthorized,
			expectedAppCode:    http.StatusUnauthorized,
			expectedMessage:    "Unauthorized: Missing claims",
		},
		{
			name:         "Internal Error - Service failure",
			callerClaims: hrClaims,
			setupMocks: func(mockSvc *mocks.MockJobGradeService) {
				mockSvc.EXPECT().ListJobGrades(gomock.Any()).Return(nil, errors.New("db failure"))
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedAppCode:    http.StatusInternalServerError,
			expectedMessage:    "Failed to retrieve job grades",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockJobGradeSvc := mocks.NewMockJobGradeService(ctrl)
			handler := NewListJobGradesHandler(mockJobGradeSvc)

			if tc.setupMocks != nil {
				tc.setupMocks(mockJobGradeSvc)
			}

			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/hr/job-grades", nil)
			c.Request = req

			if tc.callerClaims != nil {
				c.Set("claims", tc.callerClaims)
			}

			handler.ListJobGrades(c)

			require.Equal(t, tc.expectedStatusCode, recorder.Code)

			var resp common.Response
			err := json.Unmarshal(recorder.Body.Bytes(), &resp)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedAppCode, resp.Code)
			assert.Equal(t, tc.expectedMessage, resp.Message)

			if tc.expectData {
				require.NotNil(t, resp.Data, "Response data should not be nil")
			} else {
				assert.Nil(t, resp.Data, "Response data should be nil")
			}
		})
	}
}

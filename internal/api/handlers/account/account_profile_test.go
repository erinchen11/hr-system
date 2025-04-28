package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/erinchen11/hr-system/internal/interfaces/mocks"
	"github.com/erinchen11/hr-system/internal/models"
	common "github.com/erinchen11/hr-system/internal/models/common"
	"github.com/erinchen11/hr-system/internal/services"
	"github.com/erinchen11/hr-system/internal/utils"
	"github.com/gin-gonic/gin"
	gomock "github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseResponse(t *testing.T, recorder *httptest.ResponseRecorder) common.Response {
	var resp common.Response
	err := json.Unmarshal(recorder.Body.Bytes(), &resp)
	require.NoError(t, err, "Response body should be valid JSON")
	return resp
}

func TestUserProfileHandler_GetProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 通用測試資料
	testAccountID := uuid.New()
	testEmploymentID := uuid.New()
	testPhone := "123-456-7890"
	testPosition := "Software Engineer"
	testSalary := decimal.NewFromInt(70000)
	testHireDate := time.Now().AddDate(-1, -2, -3).Truncate(24 * time.Hour)

	mockAccount := &models.Account{
		ID:          testAccountID,
		FirstName:   "Profile",
		LastName:    "User",
		Email:       "profile@example.com",
		Role:        models.RoleEmployee,
		PhoneNumber: testPhone,
	}
	mockEmployment := &models.Employment{
		ID:            testEmploymentID,
		AccountID:     testAccountID,
		PositionTitle: (testPosition),
		Salary:        &testSalary,
		HireDate:      utils.Ptr(testHireDate),
		Status:        models.EmploymentStatusActive,
	}
	mockSuperAdmin := &models.Account{
		ID:        testAccountID,
		FirstName: "Super",
		LastName:  "Admin",
		Email:     "super@example.com",
		Role:      models.RoleSuperAdmin,
	}

	type testCase struct {
		name             string
		userIDInContext  interface{}
		setupMocks       func(*mocks.MockAccountService, *mocks.MockEmploymentService)
		expectStatusCode int
		expectAppCode    int
		expectMessage    string
		expectData       *UserProfileResponse
	}

	testCases := []testCase{
		{
			name:            "Success - Employee with Employment",
			userIDInContext: testAccountID.String(),
			setupMocks: func(a *mocks.MockAccountService, e *mocks.MockEmploymentService) {
				a.EXPECT().GetAccount(gomock.Any(), testAccountID).Return(mockAccount, nil)
				e.EXPECT().GetEmploymentByAccountID(gomock.Any(), testAccountID).Return(mockEmployment, nil)
			},
			expectStatusCode: http.StatusOK,
			expectAppCode:    http.StatusOK,
			expectMessage:    "Success",
			expectData: &UserProfileResponse{
				FirstName:     "Profile",
				LastName:      "User",
				Email:         "profile@example.com",
				Role:          models.RoleEmployee,
				PhoneNumber:   (testPhone),
				PositionTitle: (testPosition),
				Salary:        &testSalary,
				HireDate:      utils.Ptr(testHireDate),
				Status:        models.EmploymentStatusActive,
			},
		},
		{
			name:            "Success - SuperAdmin without Employment",
			userIDInContext: testAccountID.String(),
			setupMocks: func(a *mocks.MockAccountService, e *mocks.MockEmploymentService) {
				a.EXPECT().GetAccount(gomock.Any(), testAccountID).Return(mockSuperAdmin, nil)
				e.EXPECT().GetEmploymentByAccountID(gomock.Any(), testAccountID).Return(nil, services.ErrEmploymentNotFound)
			},
			expectStatusCode: http.StatusOK,
			expectAppCode:    http.StatusOK,
			expectMessage:    "Success",
			expectData: &UserProfileResponse{
				FirstName: "Super",
				LastName:  "Admin",
				Email:     "super@example.com",
				Role:      models.RoleSuperAdmin,
				Status:    "N/A",
			},
		},
		{
			name:             "Failure - Missing User ID",
			userIDInContext:  nil,
			expectStatusCode: http.StatusUnauthorized,
			expectAppCode:    http.StatusUnauthorized,
			expectMessage:    "Unauthorized: Missing user identity",
		},
		{
			name:             "Failure - Invalid User ID Type",
			userIDInContext:  1234,
			expectStatusCode: http.StatusInternalServerError,
			expectAppCode:    http.StatusInternalServerError,
			expectMessage:    "Internal error processing user identity",
		},
		{
			name:             "Failure - Invalid User ID Format",
			userIDInContext:  "invalid-uuid",
			expectStatusCode: http.StatusInternalServerError,
			expectAppCode:    http.StatusInternalServerError,
			expectMessage:    "Internal error processing user identity format",
		},
		{
			name:            "Failure - Account Not Found",
			userIDInContext: testAccountID.String(),
			setupMocks: func(a *mocks.MockAccountService, e *mocks.MockEmploymentService) {
				a.EXPECT().GetAccount(gomock.Any(), testAccountID).Return(nil, services.ErrAccountNotFound)
			},
			expectStatusCode: http.StatusNotFound,
			expectAppCode:    http.StatusNotFound,
			expectMessage:    "User account not found",
		},
		{
			name:            "Failure - Account Service DB Error",
			userIDInContext: testAccountID.String(),
			setupMocks: func(a *mocks.MockAccountService, e *mocks.MockEmploymentService) {
				a.EXPECT().GetAccount(gomock.Any(), testAccountID).Return(nil, errors.New("db error"))
			},
			expectStatusCode: http.StatusInternalServerError,
			expectAppCode:    http.StatusInternalServerError,
			expectMessage:    "Failed to retrieve user profile",
		},
		{
			name:            "Failure - Employment Service DB Error",
			userIDInContext: testAccountID.String(),
			setupMocks: func(a *mocks.MockAccountService, e *mocks.MockEmploymentService) {
				a.EXPECT().GetAccount(gomock.Any(), testAccountID).Return(mockAccount, nil)
				e.EXPECT().GetEmploymentByAccountID(gomock.Any(), testAccountID).Return(nil, errors.New("employment error"))
			},
			expectStatusCode: http.StatusInternalServerError,
			expectAppCode:    http.StatusInternalServerError,
			expectMessage:    "Failed to retrieve employment details",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockAccountSvc := mocks.NewMockAccountService(ctrl)
			mockEmploymentSvc := mocks.NewMockEmploymentService(ctrl)
			handler := NewUserProfileHandler(mockAccountSvc, mockEmploymentSvc)

			if tc.setupMocks != nil {
				tc.setupMocks(mockAccountSvc, mockEmploymentSvc)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/employee/profile", nil)
			c.Request = req

			if tc.userIDInContext != nil {
				c.Set("user_id", tc.userIDInContext)
			}

			handler.GetProfile(c)

			assert.Equal(t, tc.expectStatusCode, w.Code)

			resp := parseResponse(t, w)
			assert.Equal(t, tc.expectAppCode, resp.Code)
			assert.Equal(t, tc.expectMessage, resp.Message)

			if tc.expectData != nil {
				var profile UserProfileResponse
				data, _ := json.Marshal(resp.Data)
				err := json.Unmarshal(data, &profile)
				require.NoError(t, err)

				assert.Equal(t, tc.expectData.FirstName, profile.FirstName)
				assert.Equal(t, tc.expectData.LastName, profile.LastName)
				assert.Equal(t, tc.expectData.Email, profile.Email)
				assert.Equal(t, tc.expectData.Role, profile.Role)

				if tc.expectData.PhoneNumber != "" {
					assert.Equal(t, tc.expectData.PhoneNumber, profile.PhoneNumber)
				}
				if tc.expectData.PositionTitle != "" {
					assert.Equal(t, tc.expectData.PositionTitle, profile.PositionTitle)
				}
				if tc.expectData.Salary != nil {
					assert.True(t, tc.expectData.Salary.Equals(*profile.Salary))
				}
				if tc.expectData.HireDate != nil {
					assert.True(t, tc.expectData.HireDate.Equal(*profile.HireDate))
				}
				assert.Equal(t, tc.expectData.Status, profile.Status)
			} else {
				assert.Nil(t, resp.Data)
			}
		})
	}
}

// internal/api/handlers/account_create_test.go
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/erinchen11/hr-system/internal/interfaces/mocks"
	"github.com/erinchen11/hr-system/internal/models"
	common "github.com/erinchen11/hr-system/internal/models/common"
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 測試 CreateUser
func TestAccountCreationHandler_CreateUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// --- 測試使用的 Claims (模擬不同角色) ---
	superAdminClaims := &models.Claims{UserID: uuid.New().String(), Role: models.RoleSuperAdmin}
	hrClaims := &models.Claims{UserID: uuid.New().String(), Role: models.RoleHR}
	employeeClaims := &models.Claims{UserID: uuid.New().String(), Role: models.RoleEmployee}

	testCases := []struct {
		name                 string
		callerClaims         interface{} // 模擬請求的 JWT 資訊
		requestBody          string
		setupMocks           func(mockAccountSvc *mocks.MockAccountService)
		expectedStatusCode   int
		expectedResponseBody common.Response
		expectData           bool
		expectedCreatedEmail string
		expectedCreatedRole  uint8
	}{
		{
			name:         "Success - SuperAdmin creates HR",
			callerClaims: superAdminClaims,
			requestBody:  `{"first_name": "New", "last_name": "HR", "email": "newhr@example.com", "role": 1}`,
			setupMocks: func(mockAccountSvc *mocks.MockAccountService) {
				mockAccountSvc.EXPECT().
					CreateAccountWithEmployment(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, acc *models.Account, emp *models.Employment) (*models.Account, error) {
						require.Equal(t, uint8(models.RoleHR), acc.Role)
						return &models.Account{ID: uuid.New(), Email: acc.Email}, nil
					}).Times(1)
			},
			expectedStatusCode:   http.StatusCreated,
			expectedResponseBody: common.Response{Code: http.StatusCreated, Message: "HR user created successfully"},
			expectData:           true,
			expectedCreatedEmail: "newhr@example.com",
			expectedCreatedRole:  models.RoleHR,
		},
		{
			name:         "Success - SuperAdmin creates Employee",
			callerClaims: superAdminClaims,
			requestBody:  `{"first_name": "New", "last_name": "Emp", "email": "newemp@example.com", "role": 2}`,
			setupMocks: func(mockAccountSvc *mocks.MockAccountService) {
				mockAccountSvc.EXPECT().
					CreateAccountWithEmployment(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, acc *models.Account, emp *models.Employment) (*models.Account, error) {
						require.Equal(t, uint8(models.RoleEmployee), acc.Role)
						return &models.Account{ID: uuid.New(), Email: acc.Email}, nil
					}).Times(1)
			},
			expectedStatusCode:   http.StatusCreated,
			expectedResponseBody: common.Response{Code: http.StatusCreated, Message: "Employee created successfully"},
			expectData:           true,
			expectedCreatedEmail: "newemp@example.com",
			expectedCreatedRole:  models.RoleEmployee,
		},
		{
			name:         "Success - HR creates Employee",
			callerClaims: hrClaims,
			requestBody:  `{"first_name": "Another", "last_name": "Emp", "email": "anotheremp@example.com", "role": 2}`,
			setupMocks: func(mockAccountSvc *mocks.MockAccountService) {
				mockAccountSvc.EXPECT().
					CreateAccountWithEmployment(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, acc *models.Account, emp *models.Employment) (*models.Account, error) {
						require.Equal(t, uint8(models.RoleEmployee), acc.Role)
						return &models.Account{ID: uuid.New(), Email: acc.Email}, nil
					}).Times(1)
			},
			expectedStatusCode:   http.StatusCreated,
			expectedResponseBody: common.Response{Code: http.StatusCreated, Message: "Employee created successfully"},
			expectData:           true,
			expectedCreatedEmail: "anotheremp@example.com",
			expectedCreatedRole:  models.RoleEmployee,
		},
		{
			name:         "Success - SuperAdmin missing role defaults to Employee",
			callerClaims: superAdminClaims,
			requestBody:  `{"first_name": "Super", "last_name": "Default", "email": "superdefault@example.com"}`, // <== 沒 role
			setupMocks: func(mockAccountSvc *mocks.MockAccountService) {
				mockAccountSvc.EXPECT().
					CreateAccountWithEmployment(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, acc *models.Account, emp *models.Employment) (*models.Account, error) {
						require.Equal(t, uint8(models.RoleEmployee), acc.Role)
						return &models.Account{ID: uuid.New(), Email: acc.Email}, nil
					}).Times(1)
			},
			expectedStatusCode:   http.StatusCreated,
			expectedResponseBody: common.Response{Code: http.StatusCreated, Message: "Employee created successfully"},
			expectData:           true,
			expectedCreatedEmail: "superdefault@example.com",
			expectedCreatedRole:  models.RoleEmployee,
		},
		{
			name:                 "Forbidden - HR tries to create HR",
			callerClaims:         hrClaims,
			requestBody:          `{"first_name": "Cant", "last_name": "Create", "email": "cantcreate@example.com", "role": 1}`,
			expectedStatusCode:   http.StatusForbidden,
			expectedResponseBody: common.Response{Code: http.StatusForbidden, Message: "Permission denied: HR can only create Employee roles"},
		},
		{
			name:                 "Forbidden - Employee tries to create Employee",
			callerClaims:         employeeClaims,
			requestBody:          `{"first_name": "Emp", "last_name": "Try", "email": "emptry@example.com", "role": 2}`,
			expectedStatusCode:   http.StatusForbidden,
			expectedResponseBody: common.Response{Code: http.StatusForbidden, Message: "Permission denied: Insufficient privileges to create users"},
		},
		{
			name:                 "BadRequest - Invalid JSON",
			callerClaims:         superAdminClaims,
			requestBody:          `{"first_name": "Bad", "last_name": "Json", "email": badjson.com}`,
			expectedStatusCode:   http.StatusBadRequest,
			expectedResponseBody: common.Response{Code: http.StatusBadRequest, Message: "Invalid request format"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockAccountSvc := mocks.NewMockAccountService(ctrl)
			handler := NewAccountCreationHandler(mockAccountSvc)

			if tc.setupMocks != nil {
				tc.setupMocks(mockAccountSvc)
			}

			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			req, _ := http.NewRequest(http.MethodPost, "/v1/account/create", bytes.NewBufferString(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req

			if tc.callerClaims != nil {
				c.Set("claims", tc.callerClaims)
			}

			handler.CreateUser(c)

			require.Equal(t, tc.expectedStatusCode, recorder.Code)

			var actual common.Response
			err := json.Unmarshal(recorder.Body.Bytes(), &actual)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedResponseBody.Code, actual.Code)
			if tc.expectedStatusCode == http.StatusBadRequest {
				assert.Contains(t, actual.Message, "Invalid request format")
			} else {
				assert.Equal(t, tc.expectedResponseBody.Message, actual.Message)
			}

			if tc.expectData {
				require.NotNil(t, actual.Data)
				var data CreateUserResponse
				dataBytes, _ := json.Marshal(actual.Data)
				_ = json.Unmarshal(dataBytes, &data)
				assert.Equal(t, tc.expectedCreatedEmail, data.Email)
			}
		})
	}
}

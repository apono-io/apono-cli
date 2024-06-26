/*
Apono

No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)

API version: 1.0.0
*/

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package clientapi

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"reflect"
)

// InventoryAPIService InventoryAPI service
type InventoryAPIService service

type ApiListAccessBundlesRequest struct {
	ctx        context.Context
	ApiService *InventoryAPIService
	limit      *int32
	search     *string
	skip       *int32
}

func (r ApiListAccessBundlesRequest) Limit(limit int32) ApiListAccessBundlesRequest {
	r.limit = &limit
	return r
}

func (r ApiListAccessBundlesRequest) Search(search string) ApiListAccessBundlesRequest {
	r.search = &search
	return r
}

func (r ApiListAccessBundlesRequest) Skip(skip int32) ApiListAccessBundlesRequest {
	r.skip = &skip
	return r
}

func (r ApiListAccessBundlesRequest) Execute() (*PaginatedClientResponseModelBundleClientModel, *http.Response, error) {
	return r.ApiService.ListAccessBundlesExecute(r)
}

/*
ListAccessBundles List access bundles

	@param ctx context.Context - for authentication, logging, cancellation, deadlines, tracing, etc. Passed from http.Request or context.Background().
	@return ApiListAccessBundlesRequest
*/
func (a *InventoryAPIService) ListAccessBundles(ctx context.Context) ApiListAccessBundlesRequest {
	return ApiListAccessBundlesRequest{
		ApiService: a,
		ctx:        ctx,
	}
}

// Execute executes the request
//
//	@return PaginatedClientResponseModelBundleClientModel
func (a *InventoryAPIService) ListAccessBundlesExecute(r ApiListAccessBundlesRequest) (*PaginatedClientResponseModelBundleClientModel, *http.Response, error) {
	var (
		localVarHTTPMethod  = http.MethodGet
		localVarPostBody    interface{}
		formFiles           []formFile
		localVarReturnValue *PaginatedClientResponseModelBundleClientModel
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "InventoryAPIService.ListAccessBundles")
	if err != nil {
		return localVarReturnValue, nil, &GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/client/v1/inventory/bundles"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := url.Values{}
	localVarFormParams := url.Values{}

	if r.limit != nil {
		parameterAddToHeaderOrQuery(localVarQueryParams, "limit", r.limit, "")
	} else {
		var defaultValue int32 = 100
		r.limit = &defaultValue
	}
	if r.search != nil {
		parameterAddToHeaderOrQuery(localVarQueryParams, "search", r.search, "")
	}
	if r.skip != nil {
		parameterAddToHeaderOrQuery(localVarQueryParams, "skip", r.skip, "")
	} else {
		var defaultValue int32 = 0
		r.skip = &defaultValue
	}
	// to determine the Content-Type header
	localVarHTTPContentTypes := []string{}

	// set Content-Type header
	localVarHTTPContentType := selectHeaderContentType(localVarHTTPContentTypes)
	if localVarHTTPContentType != "" {
		localVarHeaderParams["Content-Type"] = localVarHTTPContentType
	}

	// to determine the Accept header
	localVarHTTPHeaderAccepts := []string{"application/json"}

	// set Accept header
	localVarHTTPHeaderAccept := selectHeaderAccept(localVarHTTPHeaderAccepts)
	if localVarHTTPHeaderAccept != "" {
		localVarHeaderParams["Accept"] = localVarHTTPHeaderAccept
	}
	req, err := a.client.prepareRequest(r.ctx, localVarPath, localVarHTTPMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, formFiles)
	if err != nil {
		return localVarReturnValue, nil, err
	}

	localVarHTTPResponse, err := a.client.callAPI(req)
	if err != nil || localVarHTTPResponse == nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	localVarBody, err := io.ReadAll(localVarHTTPResponse.Body)
	localVarHTTPResponse.Body.Close()
	localVarHTTPResponse.Body = io.NopCloser(bytes.NewBuffer(localVarBody))
	if err != nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	if localVarHTTPResponse.StatusCode >= 300 {
		newErr := &GenericOpenAPIError{
			body:  localVarBody,
			error: localVarHTTPResponse.Status,
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	err = a.client.decode(&localVarReturnValue, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
	if err != nil {
		newErr := &GenericOpenAPIError{
			body:  localVarBody,
			error: err.Error(),
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	return localVarReturnValue, localVarHTTPResponse, nil
}

type ApiListAccessUnitsRequest struct {
	ctx             context.Context
	ApiService      *InventoryAPIService
	bundleIds       *[]string
	integrationIds  *[]string
	limit           *int32
	permissionIds   *[]string
	resourceIds     *[]string
	resourceTypeIds *[]string
	search          *string
	skip            *int32
}

func (r ApiListAccessUnitsRequest) BundleIds(bundleIds []string) ApiListAccessUnitsRequest {
	r.bundleIds = &bundleIds
	return r
}

func (r ApiListAccessUnitsRequest) IntegrationIds(integrationIds []string) ApiListAccessUnitsRequest {
	r.integrationIds = &integrationIds
	return r
}

func (r ApiListAccessUnitsRequest) Limit(limit int32) ApiListAccessUnitsRequest {
	r.limit = &limit
	return r
}

func (r ApiListAccessUnitsRequest) PermissionIds(permissionIds []string) ApiListAccessUnitsRequest {
	r.permissionIds = &permissionIds
	return r
}

func (r ApiListAccessUnitsRequest) ResourceIds(resourceIds []string) ApiListAccessUnitsRequest {
	r.resourceIds = &resourceIds
	return r
}

func (r ApiListAccessUnitsRequest) ResourceTypeIds(resourceTypeIds []string) ApiListAccessUnitsRequest {
	r.resourceTypeIds = &resourceTypeIds
	return r
}

func (r ApiListAccessUnitsRequest) Search(search string) ApiListAccessUnitsRequest {
	r.search = &search
	return r
}

func (r ApiListAccessUnitsRequest) Skip(skip int32) ApiListAccessUnitsRequest {
	r.skip = &skip
	return r
}

func (r ApiListAccessUnitsRequest) Execute() (*PaginatedClientResponseModelAccessUnitClientModel, *http.Response, error) {
	return r.ApiService.ListAccessUnitsExecute(r)
}

/*
ListAccessUnits List access units

	@param ctx context.Context - for authentication, logging, cancellation, deadlines, tracing, etc. Passed from http.Request or context.Background().
	@return ApiListAccessUnitsRequest
*/
func (a *InventoryAPIService) ListAccessUnits(ctx context.Context) ApiListAccessUnitsRequest {
	return ApiListAccessUnitsRequest{
		ApiService: a,
		ctx:        ctx,
	}
}

// Execute executes the request
//
//	@return PaginatedClientResponseModelAccessUnitClientModel
func (a *InventoryAPIService) ListAccessUnitsExecute(r ApiListAccessUnitsRequest) (*PaginatedClientResponseModelAccessUnitClientModel, *http.Response, error) {
	var (
		localVarHTTPMethod  = http.MethodGet
		localVarPostBody    interface{}
		formFiles           []formFile
		localVarReturnValue *PaginatedClientResponseModelAccessUnitClientModel
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "InventoryAPIService.ListAccessUnits")
	if err != nil {
		return localVarReturnValue, nil, &GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/client/v1/inventory/access-units"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := url.Values{}
	localVarFormParams := url.Values{}

	if r.bundleIds != nil {
		t := *r.bundleIds
		if reflect.TypeOf(t).Kind() == reflect.Slice {
			s := reflect.ValueOf(t)
			for i := 0; i < s.Len(); i++ {
				parameterAddToHeaderOrQuery(localVarQueryParams, "bundle-ids", s.Index(i).Interface(), "multi")
			}
		} else {
			parameterAddToHeaderOrQuery(localVarQueryParams, "bundle-ids", t, "multi")
		}
	}
	if r.integrationIds != nil {
		t := *r.integrationIds
		if reflect.TypeOf(t).Kind() == reflect.Slice {
			s := reflect.ValueOf(t)
			for i := 0; i < s.Len(); i++ {
				parameterAddToHeaderOrQuery(localVarQueryParams, "integration-ids", s.Index(i).Interface(), "multi")
			}
		} else {
			parameterAddToHeaderOrQuery(localVarQueryParams, "integration-ids", t, "multi")
		}
	}
	if r.limit != nil {
		parameterAddToHeaderOrQuery(localVarQueryParams, "limit", r.limit, "")
	} else {
		var defaultValue int32 = 100
		r.limit = &defaultValue
	}
	if r.permissionIds != nil {
		t := *r.permissionIds
		if reflect.TypeOf(t).Kind() == reflect.Slice {
			s := reflect.ValueOf(t)
			for i := 0; i < s.Len(); i++ {
				parameterAddToHeaderOrQuery(localVarQueryParams, "permission-ids", s.Index(i).Interface(), "multi")
			}
		} else {
			parameterAddToHeaderOrQuery(localVarQueryParams, "permission-ids", t, "multi")
		}
	}
	if r.resourceIds != nil {
		t := *r.resourceIds
		if reflect.TypeOf(t).Kind() == reflect.Slice {
			s := reflect.ValueOf(t)
			for i := 0; i < s.Len(); i++ {
				parameterAddToHeaderOrQuery(localVarQueryParams, "resource-ids", s.Index(i).Interface(), "multi")
			}
		} else {
			parameterAddToHeaderOrQuery(localVarQueryParams, "resource-ids", t, "multi")
		}
	}
	if r.resourceTypeIds != nil {
		t := *r.resourceTypeIds
		if reflect.TypeOf(t).Kind() == reflect.Slice {
			s := reflect.ValueOf(t)
			for i := 0; i < s.Len(); i++ {
				parameterAddToHeaderOrQuery(localVarQueryParams, "resource-type-ids", s.Index(i).Interface(), "multi")
			}
		} else {
			parameterAddToHeaderOrQuery(localVarQueryParams, "resource-type-ids", t, "multi")
		}
	}
	if r.search != nil {
		parameterAddToHeaderOrQuery(localVarQueryParams, "search", r.search, "")
	}
	if r.skip != nil {
		parameterAddToHeaderOrQuery(localVarQueryParams, "skip", r.skip, "")
	} else {
		var defaultValue int32 = 0
		r.skip = &defaultValue
	}
	// to determine the Content-Type header
	localVarHTTPContentTypes := []string{}

	// set Content-Type header
	localVarHTTPContentType := selectHeaderContentType(localVarHTTPContentTypes)
	if localVarHTTPContentType != "" {
		localVarHeaderParams["Content-Type"] = localVarHTTPContentType
	}

	// to determine the Accept header
	localVarHTTPHeaderAccepts := []string{"application/json"}

	// set Accept header
	localVarHTTPHeaderAccept := selectHeaderAccept(localVarHTTPHeaderAccepts)
	if localVarHTTPHeaderAccept != "" {
		localVarHeaderParams["Accept"] = localVarHTTPHeaderAccept
	}
	req, err := a.client.prepareRequest(r.ctx, localVarPath, localVarHTTPMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, formFiles)
	if err != nil {
		return localVarReturnValue, nil, err
	}

	localVarHTTPResponse, err := a.client.callAPI(req)
	if err != nil || localVarHTTPResponse == nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	localVarBody, err := io.ReadAll(localVarHTTPResponse.Body)
	localVarHTTPResponse.Body.Close()
	localVarHTTPResponse.Body = io.NopCloser(bytes.NewBuffer(localVarBody))
	if err != nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	if localVarHTTPResponse.StatusCode >= 300 {
		newErr := &GenericOpenAPIError{
			body:  localVarBody,
			error: localVarHTTPResponse.Status,
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	err = a.client.decode(&localVarReturnValue, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
	if err != nil {
		newErr := &GenericOpenAPIError{
			body:  localVarBody,
			error: err.Error(),
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	return localVarReturnValue, localVarHTTPResponse, nil
}

type ApiListIntegrationRequest struct {
	ctx        context.Context
	ApiService *InventoryAPIService
	limit      *int32
	search     *string
	sessionId  *string
	skip       *int32
}

func (r ApiListIntegrationRequest) Limit(limit int32) ApiListIntegrationRequest {
	r.limit = &limit
	return r
}

func (r ApiListIntegrationRequest) Search(search string) ApiListIntegrationRequest {
	r.search = &search
	return r
}

func (r ApiListIntegrationRequest) SessionId(sessionId string) ApiListIntegrationRequest {
	r.sessionId = &sessionId
	return r
}

func (r ApiListIntegrationRequest) Skip(skip int32) ApiListIntegrationRequest {
	r.skip = &skip
	return r
}

func (r ApiListIntegrationRequest) Execute() (*PaginatedClientResponseModelIntegrationClientModel, *http.Response, error) {
	return r.ApiService.ListIntegrationExecute(r)
}

/*
ListIntegration List integrations

	@param ctx context.Context - for authentication, logging, cancellation, deadlines, tracing, etc. Passed from http.Request or context.Background().
	@return ApiListIntegrationRequest
*/
func (a *InventoryAPIService) ListIntegration(ctx context.Context) ApiListIntegrationRequest {
	return ApiListIntegrationRequest{
		ApiService: a,
		ctx:        ctx,
	}
}

// Execute executes the request
//
//	@return PaginatedClientResponseModelIntegrationClientModel
func (a *InventoryAPIService) ListIntegrationExecute(r ApiListIntegrationRequest) (*PaginatedClientResponseModelIntegrationClientModel, *http.Response, error) {
	var (
		localVarHTTPMethod  = http.MethodGet
		localVarPostBody    interface{}
		formFiles           []formFile
		localVarReturnValue *PaginatedClientResponseModelIntegrationClientModel
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "InventoryAPIService.ListIntegration")
	if err != nil {
		return localVarReturnValue, nil, &GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/client/v1/inventory/integrations"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := url.Values{}
	localVarFormParams := url.Values{}

	if r.limit != nil {
		parameterAddToHeaderOrQuery(localVarQueryParams, "limit", r.limit, "")
	} else {
		var defaultValue int32 = 100
		r.limit = &defaultValue
	}
	if r.search != nil {
		parameterAddToHeaderOrQuery(localVarQueryParams, "search", r.search, "")
	}
	if r.sessionId != nil {
		parameterAddToHeaderOrQuery(localVarQueryParams, "session_id", r.sessionId, "")
	}
	if r.skip != nil {
		parameterAddToHeaderOrQuery(localVarQueryParams, "skip", r.skip, "")
	} else {
		var defaultValue int32 = 0
		r.skip = &defaultValue
	}
	// to determine the Content-Type header
	localVarHTTPContentTypes := []string{}

	// set Content-Type header
	localVarHTTPContentType := selectHeaderContentType(localVarHTTPContentTypes)
	if localVarHTTPContentType != "" {
		localVarHeaderParams["Content-Type"] = localVarHTTPContentType
	}

	// to determine the Accept header
	localVarHTTPHeaderAccepts := []string{"application/json"}

	// set Accept header
	localVarHTTPHeaderAccept := selectHeaderAccept(localVarHTTPHeaderAccepts)
	if localVarHTTPHeaderAccept != "" {
		localVarHeaderParams["Accept"] = localVarHTTPHeaderAccept
	}
	req, err := a.client.prepareRequest(r.ctx, localVarPath, localVarHTTPMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, formFiles)
	if err != nil {
		return localVarReturnValue, nil, err
	}

	localVarHTTPResponse, err := a.client.callAPI(req)
	if err != nil || localVarHTTPResponse == nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	localVarBody, err := io.ReadAll(localVarHTTPResponse.Body)
	localVarHTTPResponse.Body.Close()
	localVarHTTPResponse.Body = io.NopCloser(bytes.NewBuffer(localVarBody))
	if err != nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	if localVarHTTPResponse.StatusCode >= 300 {
		newErr := &GenericOpenAPIError{
			body:  localVarBody,
			error: localVarHTTPResponse.Status,
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	err = a.client.decode(&localVarReturnValue, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
	if err != nil {
		newErr := &GenericOpenAPIError{
			body:  localVarBody,
			error: err.Error(),
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	return localVarReturnValue, localVarHTTPResponse, nil
}

type ApiListPermissionsRequest struct {
	ctx            context.Context
	ApiService     *InventoryAPIService
	integrationId  *string
	limit          *int32
	resourceTypeId *string
	search         *string
	sessionId      *string
	skip           *int32
}

func (r ApiListPermissionsRequest) IntegrationId(integrationId string) ApiListPermissionsRequest {
	r.integrationId = &integrationId
	return r
}

func (r ApiListPermissionsRequest) Limit(limit int32) ApiListPermissionsRequest {
	r.limit = &limit
	return r
}

func (r ApiListPermissionsRequest) ResourceTypeId(resourceTypeId string) ApiListPermissionsRequest {
	r.resourceTypeId = &resourceTypeId
	return r
}

func (r ApiListPermissionsRequest) Search(search string) ApiListPermissionsRequest {
	r.search = &search
	return r
}

func (r ApiListPermissionsRequest) SessionId(sessionId string) ApiListPermissionsRequest {
	r.sessionId = &sessionId
	return r
}

func (r ApiListPermissionsRequest) Skip(skip int32) ApiListPermissionsRequest {
	r.skip = &skip
	return r
}

func (r ApiListPermissionsRequest) Execute() (*PaginatedClientResponseModelPermissionClientModel, *http.Response, error) {
	return r.ApiService.ListPermissionsExecute(r)
}

/*
ListPermissions List permissions

	@param ctx context.Context - for authentication, logging, cancellation, deadlines, tracing, etc. Passed from http.Request or context.Background().
	@return ApiListPermissionsRequest
*/
func (a *InventoryAPIService) ListPermissions(ctx context.Context) ApiListPermissionsRequest {
	return ApiListPermissionsRequest{
		ApiService: a,
		ctx:        ctx,
	}
}

// Execute executes the request
//
//	@return PaginatedClientResponseModelPermissionClientModel
func (a *InventoryAPIService) ListPermissionsExecute(r ApiListPermissionsRequest) (*PaginatedClientResponseModelPermissionClientModel, *http.Response, error) {
	var (
		localVarHTTPMethod  = http.MethodGet
		localVarPostBody    interface{}
		formFiles           []formFile
		localVarReturnValue *PaginatedClientResponseModelPermissionClientModel
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "InventoryAPIService.ListPermissions")
	if err != nil {
		return localVarReturnValue, nil, &GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/client/v1/inventory/permissions"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := url.Values{}
	localVarFormParams := url.Values{}

	if r.integrationId != nil {
		parameterAddToHeaderOrQuery(localVarQueryParams, "integration-id", r.integrationId, "")
	}
	if r.limit != nil {
		parameterAddToHeaderOrQuery(localVarQueryParams, "limit", r.limit, "")
	} else {
		var defaultValue int32 = 100
		r.limit = &defaultValue
	}
	if r.resourceTypeId != nil {
		parameterAddToHeaderOrQuery(localVarQueryParams, "resource-type-id", r.resourceTypeId, "")
	}
	if r.search != nil {
		parameterAddToHeaderOrQuery(localVarQueryParams, "search", r.search, "")
	}
	if r.sessionId != nil {
		parameterAddToHeaderOrQuery(localVarQueryParams, "session_id", r.sessionId, "")
	}
	if r.skip != nil {
		parameterAddToHeaderOrQuery(localVarQueryParams, "skip", r.skip, "")
	} else {
		var defaultValue int32 = 0
		r.skip = &defaultValue
	}
	// to determine the Content-Type header
	localVarHTTPContentTypes := []string{}

	// set Content-Type header
	localVarHTTPContentType := selectHeaderContentType(localVarHTTPContentTypes)
	if localVarHTTPContentType != "" {
		localVarHeaderParams["Content-Type"] = localVarHTTPContentType
	}

	// to determine the Accept header
	localVarHTTPHeaderAccepts := []string{"application/json"}

	// set Accept header
	localVarHTTPHeaderAccept := selectHeaderAccept(localVarHTTPHeaderAccepts)
	if localVarHTTPHeaderAccept != "" {
		localVarHeaderParams["Accept"] = localVarHTTPHeaderAccept
	}
	req, err := a.client.prepareRequest(r.ctx, localVarPath, localVarHTTPMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, formFiles)
	if err != nil {
		return localVarReturnValue, nil, err
	}

	localVarHTTPResponse, err := a.client.callAPI(req)
	if err != nil || localVarHTTPResponse == nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	localVarBody, err := io.ReadAll(localVarHTTPResponse.Body)
	localVarHTTPResponse.Body.Close()
	localVarHTTPResponse.Body = io.NopCloser(bytes.NewBuffer(localVarBody))
	if err != nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	if localVarHTTPResponse.StatusCode >= 300 {
		newErr := &GenericOpenAPIError{
			body:  localVarBody,
			error: localVarHTTPResponse.Status,
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	err = a.client.decode(&localVarReturnValue, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
	if err != nil {
		newErr := &GenericOpenAPIError{
			body:  localVarBody,
			error: err.Error(),
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	return localVarReturnValue, localVarHTTPResponse, nil
}

type ApiListResourceTypesRequest struct {
	ctx           context.Context
	ApiService    *InventoryAPIService
	integrationId *string
	limit         *int32
	search        *string
	sessionId     *string
	skip          *int32
}

func (r ApiListResourceTypesRequest) IntegrationId(integrationId string) ApiListResourceTypesRequest {
	r.integrationId = &integrationId
	return r
}

func (r ApiListResourceTypesRequest) Limit(limit int32) ApiListResourceTypesRequest {
	r.limit = &limit
	return r
}

func (r ApiListResourceTypesRequest) Search(search string) ApiListResourceTypesRequest {
	r.search = &search
	return r
}

func (r ApiListResourceTypesRequest) SessionId(sessionId string) ApiListResourceTypesRequest {
	r.sessionId = &sessionId
	return r
}

func (r ApiListResourceTypesRequest) Skip(skip int32) ApiListResourceTypesRequest {
	r.skip = &skip
	return r
}

func (r ApiListResourceTypesRequest) Execute() (*PaginatedClientResponseModelResourceTypeClientModel, *http.Response, error) {
	return r.ApiService.ListResourceTypesExecute(r)
}

/*
ListResourceTypes List resource types

	@param ctx context.Context - for authentication, logging, cancellation, deadlines, tracing, etc. Passed from http.Request or context.Background().
	@return ApiListResourceTypesRequest
*/
func (a *InventoryAPIService) ListResourceTypes(ctx context.Context) ApiListResourceTypesRequest {
	return ApiListResourceTypesRequest{
		ApiService: a,
		ctx:        ctx,
	}
}

// Execute executes the request
//
//	@return PaginatedClientResponseModelResourceTypeClientModel
func (a *InventoryAPIService) ListResourceTypesExecute(r ApiListResourceTypesRequest) (*PaginatedClientResponseModelResourceTypeClientModel, *http.Response, error) {
	var (
		localVarHTTPMethod  = http.MethodGet
		localVarPostBody    interface{}
		formFiles           []formFile
		localVarReturnValue *PaginatedClientResponseModelResourceTypeClientModel
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "InventoryAPIService.ListResourceTypes")
	if err != nil {
		return localVarReturnValue, nil, &GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/client/v1/inventory/resource-types"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := url.Values{}
	localVarFormParams := url.Values{}

	if r.integrationId != nil {
		parameterAddToHeaderOrQuery(localVarQueryParams, "integration-id", r.integrationId, "")
	}
	if r.limit != nil {
		parameterAddToHeaderOrQuery(localVarQueryParams, "limit", r.limit, "")
	} else {
		var defaultValue int32 = 100
		r.limit = &defaultValue
	}
	if r.search != nil {
		parameterAddToHeaderOrQuery(localVarQueryParams, "search", r.search, "")
	}
	if r.sessionId != nil {
		parameterAddToHeaderOrQuery(localVarQueryParams, "session_id", r.sessionId, "")
	}
	if r.skip != nil {
		parameterAddToHeaderOrQuery(localVarQueryParams, "skip", r.skip, "")
	} else {
		var defaultValue int32 = 0
		r.skip = &defaultValue
	}
	// to determine the Content-Type header
	localVarHTTPContentTypes := []string{}

	// set Content-Type header
	localVarHTTPContentType := selectHeaderContentType(localVarHTTPContentTypes)
	if localVarHTTPContentType != "" {
		localVarHeaderParams["Content-Type"] = localVarHTTPContentType
	}

	// to determine the Accept header
	localVarHTTPHeaderAccepts := []string{"application/json"}

	// set Accept header
	localVarHTTPHeaderAccept := selectHeaderAccept(localVarHTTPHeaderAccepts)
	if localVarHTTPHeaderAccept != "" {
		localVarHeaderParams["Accept"] = localVarHTTPHeaderAccept
	}
	req, err := a.client.prepareRequest(r.ctx, localVarPath, localVarHTTPMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, formFiles)
	if err != nil {
		return localVarReturnValue, nil, err
	}

	localVarHTTPResponse, err := a.client.callAPI(req)
	if err != nil || localVarHTTPResponse == nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	localVarBody, err := io.ReadAll(localVarHTTPResponse.Body)
	localVarHTTPResponse.Body.Close()
	localVarHTTPResponse.Body = io.NopCloser(bytes.NewBuffer(localVarBody))
	if err != nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	if localVarHTTPResponse.StatusCode >= 300 {
		newErr := &GenericOpenAPIError{
			body:  localVarBody,
			error: localVarHTTPResponse.Status,
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	err = a.client.decode(&localVarReturnValue, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
	if err != nil {
		newErr := &GenericOpenAPIError{
			body:  localVarBody,
			error: err.Error(),
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	return localVarReturnValue, localVarHTTPResponse, nil
}

type ApiListResourcesRequest struct {
	ctx            context.Context
	ApiService     *InventoryAPIService
	integrationId  *string
	limit          *int32
	resourceTypeId *string
	search         *string
	sessionId      *string
	skip           *int32
	sourceId       *[]string
}

func (r ApiListResourcesRequest) IntegrationId(integrationId string) ApiListResourcesRequest {
	r.integrationId = &integrationId
	return r
}

func (r ApiListResourcesRequest) Limit(limit int32) ApiListResourcesRequest {
	r.limit = &limit
	return r
}

func (r ApiListResourcesRequest) ResourceTypeId(resourceTypeId string) ApiListResourcesRequest {
	r.resourceTypeId = &resourceTypeId
	return r
}

func (r ApiListResourcesRequest) Search(search string) ApiListResourcesRequest {
	r.search = &search
	return r
}

func (r ApiListResourcesRequest) SessionId(sessionId string) ApiListResourcesRequest {
	r.sessionId = &sessionId
	return r
}

func (r ApiListResourcesRequest) Skip(skip int32) ApiListResourcesRequest {
	r.skip = &skip
	return r
}

func (r ApiListResourcesRequest) SourceId(sourceId []string) ApiListResourcesRequest {
	r.sourceId = &sourceId
	return r
}

func (r ApiListResourcesRequest) Execute() (*PaginatedClientResponseModelResourceClientModel, *http.Response, error) {
	return r.ApiService.ListResourcesExecute(r)
}

/*
ListResources List resources

	@param ctx context.Context - for authentication, logging, cancellation, deadlines, tracing, etc. Passed from http.Request or context.Background().
	@return ApiListResourcesRequest
*/
func (a *InventoryAPIService) ListResources(ctx context.Context) ApiListResourcesRequest {
	return ApiListResourcesRequest{
		ApiService: a,
		ctx:        ctx,
	}
}

// Execute executes the request
//
//	@return PaginatedClientResponseModelResourceClientModel
func (a *InventoryAPIService) ListResourcesExecute(r ApiListResourcesRequest) (*PaginatedClientResponseModelResourceClientModel, *http.Response, error) {
	var (
		localVarHTTPMethod  = http.MethodGet
		localVarPostBody    interface{}
		formFiles           []formFile
		localVarReturnValue *PaginatedClientResponseModelResourceClientModel
	)

	localBasePath, err := a.client.cfg.ServerURLWithContext(r.ctx, "InventoryAPIService.ListResources")
	if err != nil {
		return localVarReturnValue, nil, &GenericOpenAPIError{error: err.Error()}
	}

	localVarPath := localBasePath + "/api/client/v1/inventory/resources"

	localVarHeaderParams := make(map[string]string)
	localVarQueryParams := url.Values{}
	localVarFormParams := url.Values{}

	if r.integrationId != nil {
		parameterAddToHeaderOrQuery(localVarQueryParams, "integration-id", r.integrationId, "")
	}
	if r.limit != nil {
		parameterAddToHeaderOrQuery(localVarQueryParams, "limit", r.limit, "")
	} else {
		var defaultValue int32 = 100
		r.limit = &defaultValue
	}
	if r.resourceTypeId != nil {
		parameterAddToHeaderOrQuery(localVarQueryParams, "resource-type-id", r.resourceTypeId, "")
	}
	if r.search != nil {
		parameterAddToHeaderOrQuery(localVarQueryParams, "search", r.search, "")
	}
	if r.sessionId != nil {
		parameterAddToHeaderOrQuery(localVarQueryParams, "session_id", r.sessionId, "")
	}
	if r.skip != nil {
		parameterAddToHeaderOrQuery(localVarQueryParams, "skip", r.skip, "")
	} else {
		var defaultValue int32 = 0
		r.skip = &defaultValue
	}
	if r.sourceId != nil {
		t := *r.sourceId
		if reflect.TypeOf(t).Kind() == reflect.Slice {
			s := reflect.ValueOf(t)
			for i := 0; i < s.Len(); i++ {
				parameterAddToHeaderOrQuery(localVarQueryParams, "source-id", s.Index(i).Interface(), "multi")
			}
		} else {
			parameterAddToHeaderOrQuery(localVarQueryParams, "source-id", t, "multi")
		}
	}
	// to determine the Content-Type header
	localVarHTTPContentTypes := []string{}

	// set Content-Type header
	localVarHTTPContentType := selectHeaderContentType(localVarHTTPContentTypes)
	if localVarHTTPContentType != "" {
		localVarHeaderParams["Content-Type"] = localVarHTTPContentType
	}

	// to determine the Accept header
	localVarHTTPHeaderAccepts := []string{"application/json"}

	// set Accept header
	localVarHTTPHeaderAccept := selectHeaderAccept(localVarHTTPHeaderAccepts)
	if localVarHTTPHeaderAccept != "" {
		localVarHeaderParams["Accept"] = localVarHTTPHeaderAccept
	}
	req, err := a.client.prepareRequest(r.ctx, localVarPath, localVarHTTPMethod, localVarPostBody, localVarHeaderParams, localVarQueryParams, localVarFormParams, formFiles)
	if err != nil {
		return localVarReturnValue, nil, err
	}

	localVarHTTPResponse, err := a.client.callAPI(req)
	if err != nil || localVarHTTPResponse == nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	localVarBody, err := io.ReadAll(localVarHTTPResponse.Body)
	localVarHTTPResponse.Body.Close()
	localVarHTTPResponse.Body = io.NopCloser(bytes.NewBuffer(localVarBody))
	if err != nil {
		return localVarReturnValue, localVarHTTPResponse, err
	}

	if localVarHTTPResponse.StatusCode >= 300 {
		newErr := &GenericOpenAPIError{
			body:  localVarBody,
			error: localVarHTTPResponse.Status,
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	err = a.client.decode(&localVarReturnValue, localVarBody, localVarHTTPResponse.Header.Get("Content-Type"))
	if err != nil {
		newErr := &GenericOpenAPIError{
			body:  localVarBody,
			error: err.Error(),
		}
		return localVarReturnValue, localVarHTTPResponse, newErr
	}

	return localVarReturnValue, localVarHTTPResponse, nil
}

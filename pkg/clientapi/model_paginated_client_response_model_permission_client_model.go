/*
Apono

No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)

API version: 1.0.0
*/

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package clientapi

import (
	"encoding/json"
	"fmt"
)

// checks if the PaginatedClientResponseModelPermissionClientModel type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &PaginatedClientResponseModelPermissionClientModel{}

// PaginatedClientResponseModelPermissionClientModel struct for PaginatedClientResponseModelPermissionClientModel
type PaginatedClientResponseModelPermissionClientModel struct {
	Data       []PermissionClientModel   `json:"data"`
	Pagination PaginationClientInfoModel `json:"pagination"`
}

type _PaginatedClientResponseModelPermissionClientModel PaginatedClientResponseModelPermissionClientModel

// NewPaginatedClientResponseModelPermissionClientModel instantiates a new PaginatedClientResponseModelPermissionClientModel object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewPaginatedClientResponseModelPermissionClientModel(data []PermissionClientModel, pagination PaginationClientInfoModel) *PaginatedClientResponseModelPermissionClientModel {
	this := PaginatedClientResponseModelPermissionClientModel{}
	this.Data = data
	this.Pagination = pagination
	return &this
}

// NewPaginatedClientResponseModelPermissionClientModelWithDefaults instantiates a new PaginatedClientResponseModelPermissionClientModel object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewPaginatedClientResponseModelPermissionClientModelWithDefaults() *PaginatedClientResponseModelPermissionClientModel {
	this := PaginatedClientResponseModelPermissionClientModel{}
	return &this
}

// GetData returns the Data field value
func (o *PaginatedClientResponseModelPermissionClientModel) GetData() []PermissionClientModel {
	if o == nil {
		var ret []PermissionClientModel
		return ret
	}

	return o.Data
}

// GetDataOk returns a tuple with the Data field value
// and a boolean to check if the value has been set.
func (o *PaginatedClientResponseModelPermissionClientModel) GetDataOk() ([]PermissionClientModel, bool) {
	if o == nil {
		return nil, false
	}
	return o.Data, true
}

// SetData sets field value
func (o *PaginatedClientResponseModelPermissionClientModel) SetData(v []PermissionClientModel) {
	o.Data = v
}

// GetPagination returns the Pagination field value
func (o *PaginatedClientResponseModelPermissionClientModel) GetPagination() PaginationClientInfoModel {
	if o == nil {
		var ret PaginationClientInfoModel
		return ret
	}

	return o.Pagination
}

// GetPaginationOk returns a tuple with the Pagination field value
// and a boolean to check if the value has been set.
func (o *PaginatedClientResponseModelPermissionClientModel) GetPaginationOk() (*PaginationClientInfoModel, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Pagination, true
}

// SetPagination sets field value
func (o *PaginatedClientResponseModelPermissionClientModel) SetPagination(v PaginationClientInfoModel) {
	o.Pagination = v
}

func (o PaginatedClientResponseModelPermissionClientModel) MarshalJSON() ([]byte, error) {
	toSerialize, err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o PaginatedClientResponseModelPermissionClientModel) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	toSerialize["data"] = o.Data
	toSerialize["pagination"] = o.Pagination
	return toSerialize, nil
}

func (o *PaginatedClientResponseModelPermissionClientModel) UnmarshalJSON(bytes []byte) (err error) {
	// This validates that all required properties are included in the JSON object
	// by unmarshalling the object into a generic map with string keys and checking
	// that every required field exists as a key in the generic map.
	requiredProperties := []string{
		"data",
		"pagination",
	}

	allProperties := make(map[string]interface{})

	err = json.Unmarshal(bytes, &allProperties)

	if err != nil {
		return err
	}

	for _, requiredProperty := range requiredProperties {
		if _, exists := allProperties[requiredProperty]; !exists {
			return fmt.Errorf("no value given for required property %v", requiredProperty)
		}
	}

	varPaginatedClientResponseModelPermissionClientModel := _PaginatedClientResponseModelPermissionClientModel{}

	err = json.Unmarshal(bytes, &varPaginatedClientResponseModelPermissionClientModel)

	if err != nil {
		return err
	}

	*o = PaginatedClientResponseModelPermissionClientModel(varPaginatedClientResponseModelPermissionClientModel)

	return err
}

type NullablePaginatedClientResponseModelPermissionClientModel struct {
	value *PaginatedClientResponseModelPermissionClientModel
	isSet bool
}

func (v NullablePaginatedClientResponseModelPermissionClientModel) Get() *PaginatedClientResponseModelPermissionClientModel {
	return v.value
}

func (v *NullablePaginatedClientResponseModelPermissionClientModel) Set(val *PaginatedClientResponseModelPermissionClientModel) {
	v.value = val
	v.isSet = true
}

func (v NullablePaginatedClientResponseModelPermissionClientModel) IsSet() bool {
	return v.isSet
}

func (v *NullablePaginatedClientResponseModelPermissionClientModel) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullablePaginatedClientResponseModelPermissionClientModel(val *PaginatedClientResponseModelPermissionClientModel) *NullablePaginatedClientResponseModelPermissionClientModel {
	return &NullablePaginatedClientResponseModelPermissionClientModel{value: val, isSet: true}
}

func (v NullablePaginatedClientResponseModelPermissionClientModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullablePaginatedClientResponseModelPermissionClientModel) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}

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

// checks if the PaginatedClientResponseModelResourceTypeClientModel type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &PaginatedClientResponseModelResourceTypeClientModel{}

// PaginatedClientResponseModelResourceTypeClientModel struct for PaginatedClientResponseModelResourceTypeClientModel
type PaginatedClientResponseModelResourceTypeClientModel struct {
	Data       []ResourceTypeClientModel `json:"data"`
	Pagination PaginationClientInfoModel `json:"pagination"`
}

type _PaginatedClientResponseModelResourceTypeClientModel PaginatedClientResponseModelResourceTypeClientModel

// NewPaginatedClientResponseModelResourceTypeClientModel instantiates a new PaginatedClientResponseModelResourceTypeClientModel object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewPaginatedClientResponseModelResourceTypeClientModel(data []ResourceTypeClientModel, pagination PaginationClientInfoModel) *PaginatedClientResponseModelResourceTypeClientModel {
	this := PaginatedClientResponseModelResourceTypeClientModel{}
	this.Data = data
	this.Pagination = pagination
	return &this
}

// NewPaginatedClientResponseModelResourceTypeClientModelWithDefaults instantiates a new PaginatedClientResponseModelResourceTypeClientModel object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewPaginatedClientResponseModelResourceTypeClientModelWithDefaults() *PaginatedClientResponseModelResourceTypeClientModel {
	this := PaginatedClientResponseModelResourceTypeClientModel{}
	return &this
}

// GetData returns the Data field value
func (o *PaginatedClientResponseModelResourceTypeClientModel) GetData() []ResourceTypeClientModel {
	if o == nil {
		var ret []ResourceTypeClientModel
		return ret
	}

	return o.Data
}

// GetDataOk returns a tuple with the Data field value
// and a boolean to check if the value has been set.
func (o *PaginatedClientResponseModelResourceTypeClientModel) GetDataOk() ([]ResourceTypeClientModel, bool) {
	if o == nil {
		return nil, false
	}
	return o.Data, true
}

// SetData sets field value
func (o *PaginatedClientResponseModelResourceTypeClientModel) SetData(v []ResourceTypeClientModel) {
	o.Data = v
}

// GetPagination returns the Pagination field value
func (o *PaginatedClientResponseModelResourceTypeClientModel) GetPagination() PaginationClientInfoModel {
	if o == nil {
		var ret PaginationClientInfoModel
		return ret
	}

	return o.Pagination
}

// GetPaginationOk returns a tuple with the Pagination field value
// and a boolean to check if the value has been set.
func (o *PaginatedClientResponseModelResourceTypeClientModel) GetPaginationOk() (*PaginationClientInfoModel, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Pagination, true
}

// SetPagination sets field value
func (o *PaginatedClientResponseModelResourceTypeClientModel) SetPagination(v PaginationClientInfoModel) {
	o.Pagination = v
}

func (o PaginatedClientResponseModelResourceTypeClientModel) MarshalJSON() ([]byte, error) {
	toSerialize, err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o PaginatedClientResponseModelResourceTypeClientModel) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	toSerialize["data"] = o.Data
	toSerialize["pagination"] = o.Pagination
	return toSerialize, nil
}

func (o *PaginatedClientResponseModelResourceTypeClientModel) UnmarshalJSON(bytes []byte) (err error) {
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

	varPaginatedClientResponseModelResourceTypeClientModel := _PaginatedClientResponseModelResourceTypeClientModel{}

	err = json.Unmarshal(bytes, &varPaginatedClientResponseModelResourceTypeClientModel)

	if err != nil {
		return err
	}

	*o = PaginatedClientResponseModelResourceTypeClientModel(varPaginatedClientResponseModelResourceTypeClientModel)

	return err
}

type NullablePaginatedClientResponseModelResourceTypeClientModel struct {
	value *PaginatedClientResponseModelResourceTypeClientModel
	isSet bool
}

func (v NullablePaginatedClientResponseModelResourceTypeClientModel) Get() *PaginatedClientResponseModelResourceTypeClientModel {
	return v.value
}

func (v *NullablePaginatedClientResponseModelResourceTypeClientModel) Set(val *PaginatedClientResponseModelResourceTypeClientModel) {
	v.value = val
	v.isSet = true
}

func (v NullablePaginatedClientResponseModelResourceTypeClientModel) IsSet() bool {
	return v.isSet
}

func (v *NullablePaginatedClientResponseModelResourceTypeClientModel) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullablePaginatedClientResponseModelResourceTypeClientModel(val *PaginatedClientResponseModelResourceTypeClientModel) *NullablePaginatedClientResponseModelResourceTypeClientModel {
	return &NullablePaginatedClientResponseModelResourceTypeClientModel{value: val, isSet: true}
}

func (v NullablePaginatedClientResponseModelResourceTypeClientModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullablePaginatedClientResponseModelResourceTypeClientModel) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
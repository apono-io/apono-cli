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

// checks if the PaginatedClientResponseModelAccessRequestClientModel type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &PaginatedClientResponseModelAccessRequestClientModel{}

// PaginatedClientResponseModelAccessRequestClientModel struct for PaginatedClientResponseModelAccessRequestClientModel
type PaginatedClientResponseModelAccessRequestClientModel struct {
	Data       []AccessRequestClientModel `json:"data"`
	Pagination PaginationClientInfoModel  `json:"pagination"`
}

type _PaginatedClientResponseModelAccessRequestClientModel PaginatedClientResponseModelAccessRequestClientModel

// NewPaginatedClientResponseModelAccessRequestClientModel instantiates a new PaginatedClientResponseModelAccessRequestClientModel object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewPaginatedClientResponseModelAccessRequestClientModel(data []AccessRequestClientModel, pagination PaginationClientInfoModel) *PaginatedClientResponseModelAccessRequestClientModel {
	this := PaginatedClientResponseModelAccessRequestClientModel{}
	this.Data = data
	this.Pagination = pagination
	return &this
}

// NewPaginatedClientResponseModelAccessRequestClientModelWithDefaults instantiates a new PaginatedClientResponseModelAccessRequestClientModel object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewPaginatedClientResponseModelAccessRequestClientModelWithDefaults() *PaginatedClientResponseModelAccessRequestClientModel {
	this := PaginatedClientResponseModelAccessRequestClientModel{}
	return &this
}

// GetData returns the Data field value
func (o *PaginatedClientResponseModelAccessRequestClientModel) GetData() []AccessRequestClientModel {
	if o == nil {
		var ret []AccessRequestClientModel
		return ret
	}

	return o.Data
}

// GetDataOk returns a tuple with the Data field value
// and a boolean to check if the value has been set.
func (o *PaginatedClientResponseModelAccessRequestClientModel) GetDataOk() ([]AccessRequestClientModel, bool) {
	if o == nil {
		return nil, false
	}
	return o.Data, true
}

// SetData sets field value
func (o *PaginatedClientResponseModelAccessRequestClientModel) SetData(v []AccessRequestClientModel) {
	o.Data = v
}

// GetPagination returns the Pagination field value
func (o *PaginatedClientResponseModelAccessRequestClientModel) GetPagination() PaginationClientInfoModel {
	if o == nil {
		var ret PaginationClientInfoModel
		return ret
	}

	return o.Pagination
}

// GetPaginationOk returns a tuple with the Pagination field value
// and a boolean to check if the value has been set.
func (o *PaginatedClientResponseModelAccessRequestClientModel) GetPaginationOk() (*PaginationClientInfoModel, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Pagination, true
}

// SetPagination sets field value
func (o *PaginatedClientResponseModelAccessRequestClientModel) SetPagination(v PaginationClientInfoModel) {
	o.Pagination = v
}

func (o PaginatedClientResponseModelAccessRequestClientModel) MarshalJSON() ([]byte, error) {
	toSerialize, err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o PaginatedClientResponseModelAccessRequestClientModel) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	toSerialize["data"] = o.Data
	toSerialize["pagination"] = o.Pagination
	return toSerialize, nil
}

func (o *PaginatedClientResponseModelAccessRequestClientModel) UnmarshalJSON(bytes []byte) (err error) {
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

	varPaginatedClientResponseModelAccessRequestClientModel := _PaginatedClientResponseModelAccessRequestClientModel{}

	err = json.Unmarshal(bytes, &varPaginatedClientResponseModelAccessRequestClientModel)

	if err != nil {
		return err
	}

	*o = PaginatedClientResponseModelAccessRequestClientModel(varPaginatedClientResponseModelAccessRequestClientModel)

	return err
}

type NullablePaginatedClientResponseModelAccessRequestClientModel struct {
	value *PaginatedClientResponseModelAccessRequestClientModel
	isSet bool
}

func (v NullablePaginatedClientResponseModelAccessRequestClientModel) Get() *PaginatedClientResponseModelAccessRequestClientModel {
	return v.value
}

func (v *NullablePaginatedClientResponseModelAccessRequestClientModel) Set(val *PaginatedClientResponseModelAccessRequestClientModel) {
	v.value = val
	v.isSet = true
}

func (v NullablePaginatedClientResponseModelAccessRequestClientModel) IsSet() bool {
	return v.isSet
}

func (v *NullablePaginatedClientResponseModelAccessRequestClientModel) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullablePaginatedClientResponseModelAccessRequestClientModel(val *PaginatedClientResponseModelAccessRequestClientModel) *NullablePaginatedClientResponseModelAccessRequestClientModel {
	return &NullablePaginatedClientResponseModelAccessRequestClientModel{value: val, isSet: true}
}

func (v NullablePaginatedClientResponseModelAccessRequestClientModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullablePaginatedClientResponseModelAccessRequestClientModel) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}

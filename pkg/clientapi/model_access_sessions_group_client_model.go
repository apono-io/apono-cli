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

// checks if the AccessSessionsGroupClientModel type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &AccessSessionsGroupClientModel{}

// AccessSessionsGroupClientModel struct for AccessSessionsGroupClientModel
type AccessSessionsGroupClientModel struct {
	Integration NullableAccessSessionsGroupClientModelIntegration `json:"integration,omitempty"`
	Total       int32                                             `json:"total"`
	Sessions    []AccessSessionClientModel                        `json:"sessions"`
}

type _AccessSessionsGroupClientModel AccessSessionsGroupClientModel

// NewAccessSessionsGroupClientModel instantiates a new AccessSessionsGroupClientModel object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewAccessSessionsGroupClientModel(total int32, sessions []AccessSessionClientModel) *AccessSessionsGroupClientModel {
	this := AccessSessionsGroupClientModel{}
	this.Total = total
	this.Sessions = sessions
	return &this
}

// NewAccessSessionsGroupClientModelWithDefaults instantiates a new AccessSessionsGroupClientModel object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewAccessSessionsGroupClientModelWithDefaults() *AccessSessionsGroupClientModel {
	this := AccessSessionsGroupClientModel{}
	return &this
}

// GetIntegration returns the Integration field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *AccessSessionsGroupClientModel) GetIntegration() AccessSessionsGroupClientModelIntegration {
	if o == nil || IsNil(o.Integration.Get()) {
		var ret AccessSessionsGroupClientModelIntegration
		return ret
	}
	return *o.Integration.Get()
}

// GetIntegrationOk returns a tuple with the Integration field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *AccessSessionsGroupClientModel) GetIntegrationOk() (*AccessSessionsGroupClientModelIntegration, bool) {
	if o == nil {
		return nil, false
	}
	return o.Integration.Get(), o.Integration.IsSet()
}

// HasIntegration returns a boolean if a field has been set.
func (o *AccessSessionsGroupClientModel) HasIntegration() bool {
	if o != nil && o.Integration.IsSet() {
		return true
	}

	return false
}

// SetIntegration gets a reference to the given NullableAccessSessionsGroupClientModelIntegration and assigns it to the Integration field.
func (o *AccessSessionsGroupClientModel) SetIntegration(v AccessSessionsGroupClientModelIntegration) {
	o.Integration.Set(&v)
}

// SetIntegrationNil sets the value for Integration to be an explicit nil
func (o *AccessSessionsGroupClientModel) SetIntegrationNil() {
	o.Integration.Set(nil)
}

// UnsetIntegration ensures that no value is present for Integration, not even an explicit nil
func (o *AccessSessionsGroupClientModel) UnsetIntegration() {
	o.Integration.Unset()
}

// GetTotal returns the Total field value
func (o *AccessSessionsGroupClientModel) GetTotal() int32 {
	if o == nil {
		var ret int32
		return ret
	}

	return o.Total
}

// GetTotalOk returns a tuple with the Total field value
// and a boolean to check if the value has been set.
func (o *AccessSessionsGroupClientModel) GetTotalOk() (*int32, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Total, true
}

// SetTotal sets field value
func (o *AccessSessionsGroupClientModel) SetTotal(v int32) {
	o.Total = v
}

// GetSessions returns the Sessions field value
func (o *AccessSessionsGroupClientModel) GetSessions() []AccessSessionClientModel {
	if o == nil {
		var ret []AccessSessionClientModel
		return ret
	}

	return o.Sessions
}

// GetSessionsOk returns a tuple with the Sessions field value
// and a boolean to check if the value has been set.
func (o *AccessSessionsGroupClientModel) GetSessionsOk() ([]AccessSessionClientModel, bool) {
	if o == nil {
		return nil, false
	}
	return o.Sessions, true
}

// SetSessions sets field value
func (o *AccessSessionsGroupClientModel) SetSessions(v []AccessSessionClientModel) {
	o.Sessions = v
}

func (o AccessSessionsGroupClientModel) MarshalJSON() ([]byte, error) {
	toSerialize, err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o AccessSessionsGroupClientModel) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	if o.Integration.IsSet() {
		toSerialize["integration"] = o.Integration.Get()
	}
	toSerialize["total"] = o.Total
	toSerialize["sessions"] = o.Sessions
	return toSerialize, nil
}

func (o *AccessSessionsGroupClientModel) UnmarshalJSON(bytes []byte) (err error) {
	// This validates that all required properties are included in the JSON object
	// by unmarshalling the object into a generic map with string keys and checking
	// that every required field exists as a key in the generic map.
	requiredProperties := []string{
		"total",
		"sessions",
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

	varAccessSessionsGroupClientModel := _AccessSessionsGroupClientModel{}

	err = json.Unmarshal(bytes, &varAccessSessionsGroupClientModel)

	if err != nil {
		return err
	}

	*o = AccessSessionsGroupClientModel(varAccessSessionsGroupClientModel)

	return err
}

type NullableAccessSessionsGroupClientModel struct {
	value *AccessSessionsGroupClientModel
	isSet bool
}

func (v NullableAccessSessionsGroupClientModel) Get() *AccessSessionsGroupClientModel {
	return v.value
}

func (v *NullableAccessSessionsGroupClientModel) Set(val *AccessSessionsGroupClientModel) {
	v.value = val
	v.isSet = true
}

func (v NullableAccessSessionsGroupClientModel) IsSet() bool {
	return v.isSet
}

func (v *NullableAccessSessionsGroupClientModel) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableAccessSessionsGroupClientModel(val *AccessSessionsGroupClientModel) *NullableAccessSessionsGroupClientModel {
	return &NullableAccessSessionsGroupClientModel{value: val, isSet: true}
}

func (v NullableAccessSessionsGroupClientModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableAccessSessionsGroupClientModel) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}

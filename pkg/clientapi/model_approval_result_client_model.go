/*
Apono

No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)

API version: 1.0.0
*/

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package clientapi

import (
	"encoding/json"
)

// checks if the ApprovalResultClientModel type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &ApprovalResultClientModel{}

// ApprovalResultClientModel struct for ApprovalResultClientModel
type ApprovalResultClientModel struct {
	Justification NullableString `json:"justification,omitempty"`
}

// NewApprovalResultClientModel instantiates a new ApprovalResultClientModel object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewApprovalResultClientModel() *ApprovalResultClientModel {
	this := ApprovalResultClientModel{}
	return &this
}

// NewApprovalResultClientModelWithDefaults instantiates a new ApprovalResultClientModel object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewApprovalResultClientModelWithDefaults() *ApprovalResultClientModel {
	this := ApprovalResultClientModel{}
	return &this
}

// GetJustification returns the Justification field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *ApprovalResultClientModel) GetJustification() string {
	if o == nil || IsNil(o.Justification.Get()) {
		var ret string
		return ret
	}
	return *o.Justification.Get()
}

// GetJustificationOk returns a tuple with the Justification field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *ApprovalResultClientModel) GetJustificationOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.Justification.Get(), o.Justification.IsSet()
}

// HasJustification returns a boolean if a field has been set.
func (o *ApprovalResultClientModel) HasJustification() bool {
	if o != nil && o.Justification.IsSet() {
		return true
	}

	return false
}

// SetJustification gets a reference to the given NullableString and assigns it to the Justification field.
func (o *ApprovalResultClientModel) SetJustification(v string) {
	o.Justification.Set(&v)
}

// SetJustificationNil sets the value for Justification to be an explicit nil
func (o *ApprovalResultClientModel) SetJustificationNil() {
	o.Justification.Set(nil)
}

// UnsetJustification ensures that no value is present for Justification, not even an explicit nil
func (o *ApprovalResultClientModel) UnsetJustification() {
	o.Justification.Unset()
}

func (o ApprovalResultClientModel) MarshalJSON() ([]byte, error) {
	toSerialize, err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o ApprovalResultClientModel) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	if o.Justification.IsSet() {
		toSerialize["justification"] = o.Justification.Get()
	}
	return toSerialize, nil
}

type NullableApprovalResultClientModel struct {
	value *ApprovalResultClientModel
	isSet bool
}

func (v NullableApprovalResultClientModel) Get() *ApprovalResultClientModel {
	return v.value
}

func (v *NullableApprovalResultClientModel) Set(val *ApprovalResultClientModel) {
	v.value = val
	v.isSet = true
}

func (v NullableApprovalResultClientModel) IsSet() bool {
	return v.isSet
}

func (v *NullableApprovalResultClientModel) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableApprovalResultClientModel(val *ApprovalResultClientModel) *NullableApprovalResultClientModel {
	return &NullableApprovalResultClientModel{value: val, isSet: true}
}

func (v NullableApprovalResultClientModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableApprovalResultClientModel) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}